package lab

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sort"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"

	"github.com/Chocapikk/pik/pkg/output"
	"github.com/Chocapikk/pik/pkg/text"
	"github.com/Chocapikk/pik/sdk"
)

const (
	labelLab     = "pik.lab"
	labelService = "pik.lab.service"
)

// manager implements sdk.LabManager.
type manager struct{}

func init() {
	sdk.SetLabManager(&manager{})
}

func (m *manager) Start(ctx context.Context, name string, services []sdk.Service) error {
	return Start(ctx, name, services)
}
func (m *manager) Stop(ctx context.Context, name string) error {
	return Stop(ctx, name)
}
func (m *manager) Status(ctx context.Context) ([]sdk.LabStatus, error) {
	labs, err := Status(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]sdk.LabStatus, len(labs))
	for i, l := range labs {
		svcs := make([]sdk.LabServiceStatus, len(l.Services))
		for j, s := range l.Services {
			svcs[j] = sdk.LabServiceStatus{Name: s.Name, Image: s.Image, State: s.State, Ports: s.Ports}
		}
		result[i] = sdk.LabStatus{Name: l.Name, Services: svcs}
	}
	return result, nil
}
func (m *manager) IsRunning(ctx context.Context, name string) bool {
	return IsRunning(ctx, name)
}
func (m *manager) Target(ctx context.Context, name string) string {
	return Target(ctx, name)
}
func (m *manager) WaitReady(ctx context.Context, addr string, timeout time.Duration) error {
	return WaitReady(ctx, addr, timeout)
}
func (m *manager) WaitProbe(ctx context.Context, timeout time.Duration, fn func() error) error {
	return WaitProbe(ctx, timeout, fn)
}
func (m *manager) DockerGateway() string {
	return DockerGateway()
}

// --- Docker client ---

func withClient(fn func(cli *client.Client) error) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("docker not available: %w", err)
	}
	defer cli.Close()
	return fn(cli)
}

// --- Poll loop ---

// poll retries fn every interval until it returns nil or the deadline passes.
func poll(ctx context.Context, timeout, interval time.Duration, fn func() error) error {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		lastErr = fn()
		if lastErr == nil {
			return nil
		}
		time.Sleep(interval)
	}
	return lastErr
}

// --- Public API ---

// Start creates and starts containers for the given lab services.
//
// Each service gets a network alias matching its Name so services
// can reference each other by short name (like docker compose).
// Services with a healthcheck are waited on before starting the
// next service, giving dependencies time to become ready.
func Start(ctx context.Context, name string, services []sdk.Service) error {
	if len(services) == 0 {
		return fmt.Errorf("lab has no services")
	}

	return withClient(func(cli *client.Client) error {
		teardown(ctx, cli, name)

		// Shared random values so services in a lab resolve the same
		// sdk.Rand("label") to the same generated value.
		randoms := make(map[string]string)

		netName := "pik_" + name
		netResp, err := cli.NetworkCreate(ctx, netName, network.CreateOptions{
			Labels: map[string]string{labelLab: name},
		})
		if err != nil {
			return fmt.Errorf("create network: %w", err)
		}

		for _, svc := range services {
			if svc.Image == "" {
				return fmt.Errorf("service %q: no image specified", svc.Name)
			}

			output.Status("Pulling %s", svc.Image)
			reader, err := cli.ImagePull(ctx, svc.Image, image.PullOptions{})
			if err != nil {
				return fmt.Errorf("pull %s: %w", svc.Image, err)
			}
			if err := showPullProgress(reader); err != nil {
				return fmt.Errorf("pull %s: %w", svc.Image, err)
			}

			cfg, hostCfg := toDocker(svc, name, randoms)

			netCfg := &network.NetworkingConfig{
				EndpointsConfig: map[string]*network.EndpointSettings{
					netName: {
						NetworkID: netResp.ID,
						Aliases:   []string{svc.Name},
					},
				},
			}

			containerName := "pik-" + name + "-" + svc.Name
			output.Status("Creating %s", containerName)
			resp, err := cli.ContainerCreate(ctx, cfg, hostCfg, netCfg, nil, containerName)
			if err != nil {
				return fmt.Errorf("create %s: %w", containerName, err)
			}

			if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
				return fmt.Errorf("start %s: %w", containerName, err)
			}
			output.Success("Started %s (%s)", containerName, svc.Image)

			if len(svc.Healthcheck) > 0 {
				output.Status("Waiting for %s to be healthy", svc.Name)
				err := poll(ctx, 120*time.Second, 2*time.Second, func() error {
					inspect, err := cli.ContainerInspect(ctx, resp.ID)
					if err != nil {
						return err
					}
					if inspect.State.Health == nil {
						return fmt.Errorf("no health status yet")
					}
					switch inspect.State.Health.Status {
					case "healthy":
						return nil
					case "unhealthy":
						return fmt.Errorf("container unhealthy")
					}
					return fmt.Errorf("health: %s", inspect.State.Health.Status)
				})
				if err != nil {
					output.Warning("%s: %v", svc.Name, err)
				}
			}
		}
		return nil
	})
}

// Stop tears down all containers and the network for a lab.
func Stop(ctx context.Context, name string) error {
	return withClient(func(cli *client.Client) error {
		teardown(ctx, cli, name)
		return nil
	})
}

// --- Status ---

// LabInfo holds status for a lab.
type LabInfo struct {
	Name     string
	Services []ServiceInfo
}

// ServiceInfo holds status for one service container.
type ServiceInfo struct {
	Name  string
	Image string
	State string
	Ports string
}

// Status returns info for all labs tracked by pik.
func Status(ctx context.Context) ([]LabInfo, error) {
	var labs []LabInfo
	err := withClient(func(cli *client.Client) error {
		containers, err := cli.ContainerList(ctx, container.ListOptions{
			All:     true,
			Filters: filters.NewArgs(filters.Arg("label", labelLab)),
		})
		if err != nil {
			return err
		}
		if len(containers) == 0 {
			return nil
		}

		byLab := make(map[string][]ServiceInfo)
		for _, ctr := range containers {
			byLab[ctr.Labels[labelLab]] = append(byLab[ctr.Labels[labelLab]], ServiceInfo{
				Name:  ctr.Labels[labelService],
				Image: ctr.Image,
				State: ctr.State,
				Ports: formatPorts(ctr.Ports),
			})
		}
		for labName, services := range byLab {
			labs = append(labs, LabInfo{Name: labName, Services: services})
		}
		sort.Slice(labs, func(i, j int) bool { return labs[i].Name < labs[j].Name })
		return nil
	})
	return labs, err
}

// IsRunning checks if a lab has running containers.
func IsRunning(ctx context.Context, name string) bool {
	running := false
	withClient(func(cli *client.Client) error {
		containers, err := cli.ContainerList(ctx, container.ListOptions{
			Filters: filters.NewArgs(
				filters.Arg("label", labelLab+"="+name),
				filters.Arg("status", "running"),
			),
		})
		running = err == nil && len(containers) > 0
		return nil
	})
	return running
}

// --- Readiness ---

// Target queries Docker for the actual mapped port of a running lab.
// Since labs use random host ports, we can't read from static config.
func Target(ctx context.Context, name string) string {
	target := "127.0.0.1"
	withClient(func(cli *client.Client) error {
		containers, err := cli.ContainerList(ctx, container.ListOptions{
			Filters: labFilter(name),
		})
		if err != nil || len(containers) == 0 {
			return nil
		}
		// Find the first container with a mapped port.
		for _, ctr := range containers {
			for _, p := range ctr.Ports {
				if p.PublicPort > 0 {
					target = fmt.Sprintf("127.0.0.1:%d", p.PublicPort)
					return nil
				}
			}
		}
		return nil
	})
	return target
}

// WaitReady polls a TCP address until it accepts connections.
func WaitReady(ctx context.Context, addr string, timeout time.Duration) error {
	return poll(ctx, timeout, 2*time.Second, func() error {
		conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
		if err != nil {
			return err
		}
		conn.Close()
		return nil
	})
}

// WaitProbe retries fn until it returns nil or timeout expires.
// Use with the module's Check() as fn to wait for the application
// layer to be fully ready (not just TCP).
func WaitProbe(ctx context.Context, timeout time.Duration, fn func() error) error {
	return poll(ctx, timeout, 3*time.Second, fn)
}

// DockerGateway returns the default Docker bridge gateway IP.
func DockerGateway() string {
	conn, err := net.DialTimeout("tcp", "172.17.0.1:1", 100*time.Millisecond)
	if conn != nil {
		conn.Close()
	}
	if err != nil && strings.Contains(err.Error(), "no route") {
		return ""
	}
	return "172.17.0.1"
}

// --- Internal ---

// toDocker converts an sdk.Service to Docker SDK types.
func toDocker(svc sdk.Service, labName string, randoms map[string]string) (*container.Config, *container.HostConfig) {
	exposed, bindings, _ := nat.ParsePortSpecs(svc.Ports)

	cfg := &container.Config{
		Image:        svc.Image,
		ExposedPorts: exposed,
		Env:          envSlice(svc.Env, randoms),
		Cmd:          svc.Cmd,
		Labels: map[string]string{
			labelLab:     labName,
			labelService: svc.Name,
		},
	}
	if len(svc.Healthcheck) > 0 {
		cfg.Healthcheck = &container.HealthConfig{
			Test:        append([]string{"CMD-SHELL"}, svc.Healthcheck...),
			Interval:    5 * time.Second,
			StartPeriod: 30 * time.Second,
		}
	}

	// Force localhost + random host port. Avoids port conflicts
	// and never exposes labs to the network.
	for port, portBindings := range bindings {
		for i := range portBindings {
			portBindings[i].HostIP = "127.0.0.1"
			portBindings[i].HostPort = "0"
		}
		bindings[port] = portBindings
	}

	hostCfg := &container.HostConfig{
		PortBindings:  bindings,
		Binds:         svc.Volumes,
		RestartPolicy: container.RestartPolicy{Name: "unless-stopped"},
	}

	return cfg, hostCfg
}

// showPullProgress reads the Docker image pull JSON stream and prints
// status updates. Drains the reader fully so the pull completes.
func showPullProgress(reader io.ReadCloser) error {
	defer reader.Close()

	type pullMsg struct {
		Status string `json:"status"`
		ID     string `json:"id"`
		Error  string `json:"error"`
	}

	dec := json.NewDecoder(reader)
	var last string
	for dec.More() {
		var msg pullMsg
		if err := dec.Decode(&msg); err != nil {
			break
		}
		if msg.Error != "" {
			return fmt.Errorf("%s", msg.Error)
		}
		// Only print meaningful status changes, not per-layer progress.
		line := msg.Status
		if msg.ID != "" {
			line = msg.ID + ": " + msg.Status
		}
		if line != last && msg.Status != "" {
			// Show final status lines (Pull complete, Digest, Status).
			if msg.ID == "" || msg.Status == "Pull complete" || msg.Status == "Already exists" {
				output.Status("%s", line)
			}
			last = line
		}
	}
	return nil
}

// envSlice converts env map to Docker format, resolving sdk.Rand()
// placeholders to random values. Same label = same value across services.
func envSlice(m map[string]string, randoms map[string]string) []string {
	if len(m) == 0 {
		return nil
	}
	env := make([]string, 0, len(m))
	for k, v := range m {
		if label, ok := sdk.IsRand(v); ok {
			if resolved, exists := randoms[label]; exists {
				v = resolved
			} else {
				v = text.RandAlphaNum(16)
				randoms[label] = v
			}
		}
		env = append(env, k+"="+v)
	}
	return env
}

func labFilter(name string) filters.Args {
	return labFilter(name)
}

func teardown(ctx context.Context, cli *client.Client, name string) {
	containers, _ := cli.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: labFilter(name),
	})
	for _, ctr := range containers {
		cli.ContainerStop(ctx, ctr.ID, container.StopOptions{})
		cli.ContainerRemove(ctx, ctr.ID, container.RemoveOptions{Force: true})
	}

	networks, _ := cli.NetworkList(ctx, network.ListOptions{
		Filters: labFilter(name),
	})
	for _, n := range networks {
		cli.NetworkRemove(ctx, n.ID)
	}
}

func formatPorts(ports []container.Port) string {
	var parts []string
	for _, p := range ports {
		if p.PublicPort > 0 {
			parts = append(parts, fmt.Sprintf("%d->%d/%s", p.PublicPort, p.PrivatePort, p.Type))
		}
	}
	return strings.Join(parts, ", ")
}
