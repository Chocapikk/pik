package payload

// GenerateFunc is a function that generates a payload command string.
type GenerateFunc func(lhost string, lport int) string

// Info describes a registered payload.
type Info struct {
	Name        string
	Description string
	Platform    string // "linux", "windows"
	Generate    GenerateFunc
}

var payloads []*Info

func reg(name, desc, platform string, gen GenerateFunc) {
	payloads = append(payloads, &Info{
		Name:        name,
		Description: desc,
		Platform:    platform,
		Generate:    gen,
	})
}

func init() {
	// Linux - Bash
	reg("cmd/bash/reverse_tcp", "Bash /dev/tcp reverse shell", "linux", Bash)
	reg("cmd/bash/reverse_tcp_min", "Minimal sh /dev/tcp reverse shell", "linux", BashMin)
	reg("cmd/bash/reverse_fd", "Bash file descriptor reverse shell", "linux", BashFD)
	reg("cmd/bash/reverse_readline", "Bash readline reverse shell", "linux", BashReadLine)

	// Linux - Python
	reg("cmd/python/reverse_tcp", "Python3 reverse shell", "linux", Python)
	reg("cmd/python/reverse_tcp_min", "Compact Python3 reverse shell", "linux", PythonMin)
	reg("cmd/python/reverse_tcp_pty", "Python3 PTY reverse shell", "linux", PythonPTY)

	// Linux - Scripting languages
	reg("cmd/perl/reverse_tcp", "Perl reverse shell", "linux", Perl)
	reg("cmd/ruby/reverse_tcp", "Ruby reverse shell", "linux", Ruby)
	reg("cmd/lua/reverse_tcp", "Lua reverse shell", "linux", Lua)
	reg("cmd/nodejs/reverse_tcp", "Node.js reverse shell", "linux", NodeJS)
	reg("cmd/awk/reverse_tcp", "Awk reverse shell", "linux", Awk)

	// Linux - PHP
	reg("cmd/php/reverse_tcp", "PHP reverse shell", "linux", PHP)
	reg("cmd/php/reverse_tcp_min", "Minimal PHP reverse shell", "linux", PHPMin)
	reg("cmd/php/reverse_tcp_exec", "PHP proc_open reverse shell", "linux", PHPExec)

	// Linux - Netcat
	reg("cmd/netcat/reverse_tcp", "Netcat -e reverse shell", "linux", Netcat)
	reg("cmd/netcat/reverse_mkfifo", "Netcat mkfifo reverse shell", "linux", NetcatMkfifo)
	reg("cmd/netcat/reverse_openbsd", "OpenBSD netcat reverse shell", "linux", NetcatOpenbsd)

	// Linux - Other
	reg("cmd/socat/reverse_tty", "Socat TTY reverse shell", "linux", Socat)
	reg("cmd/java/reverse_tcp", "Java Runtime reverse shell", "linux", Java)

	// Linux - TLS
	reg("cmd/bash/reverse_tls", "Bash openssl TLS reverse shell", "linux", BashTLS)
	reg("cmd/python/reverse_tls", "Python3 TLS reverse shell", "linux", PythonTLS)
	reg("cmd/ncat/reverse_tls", "Ncat TLS reverse shell", "linux", NcatTLS)
	reg("cmd/socat/reverse_tls", "Socat TLS reverse shell", "linux", SocatTLS)

	// Linux - HTTP
	reg("cmd/curl/reverse_http", "Curl HTTP polling reverse shell", "linux", CurlHTTP)
	reg("cmd/wget/reverse_http", "Wget HTTP polling reverse shell", "linux", WgetHTTP)
	reg("cmd/php/reverse_http", "PHP HTTP polling reverse shell", "linux", PHPHTTP)
	reg("cmd/python/reverse_http", "Python3 HTTP polling reverse shell", "linux", PythonHTTP)

	// Windows
	reg("cmd/powershell/reverse_tcp", "PowerShell reverse shell", "windows", PowerShell)
	reg("cmd/powershell/reverse_conpty", "PowerShell ConPTY reverse shell", "windows", PowerShellConPTY)
}

// ListPayloads returns all registered payloads.
func ListPayloads() []*Info {
	return payloads
}

// ListForPlatform returns payloads compatible with the given platform.
func ListForPlatform(platform string) []*Info {
	var result []*Info
	for _, pl := range payloads {
		if pl.Platform == platform || platform == "" {
			result = append(result, pl)
		}
	}
	return result
}

// GetPayload returns a payload by name, or nil if not found.
func GetPayload(name string) *Info {
	for _, pl := range payloads {
		if pl.Name == name {
			return pl
		}
	}
	return nil
}

// Names returns all registered payload names.
func Names() []string {
	result := make([]string, len(payloads))
	for i, pl := range payloads {
		result[i] = pl.Name
	}
	return result
}

// DefaultPayload returns the default payload for a platform.
func DefaultPayload(platform string) *Info {
	switch platform {
	case "windows":
		return GetPayload("cmd/powershell/reverse_tcp")
	default:
		return GetPayload("cmd/bash/reverse_tcp")
	}
}
