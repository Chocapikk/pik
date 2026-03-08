package payload

import "github.com/Chocapikk/pik/sdk"

func init() {
	// Linux encoders
	sdk.RegisterEncoder(&sdk.Encoder{Name: "cmd/base64", Platform: "linux", Desc: "Base64 pipe to bash", Fn: Base64Bash})
	sdk.RegisterEncoder(&sdk.Encoder{Name: "cmd/base64_sub", Platform: "linux", Desc: "Base64 via bash -c subst", Fn: Base64BashC})
	sdk.RegisterEncoder(&sdk.Encoder{Name: "cmd/hex", Platform: "linux", Desc: "Hex pipe to bash via xxd", Fn: HexBash})
	sdk.RegisterEncoder(&sdk.Encoder{Name: "cmd/python", Platform: "linux", Desc: "Base64 via python3 os.system", Fn: Base64Python})
	sdk.RegisterEncoder(&sdk.Encoder{Name: "cmd/perl", Platform: "linux", Desc: "Base64 via perl MIME::Base64", Fn: Base64Perl})
	sdk.RegisterEncoder(&sdk.Encoder{Name: "cmd/php", Platform: "linux", Desc: "Base64 via php system()", Fn: func(cmd string) string {
		return Wrap(cmd, Base64Enc, PHPDec)
	}})
	sdk.RegisterEncoder(&sdk.Encoder{Name: "cmd/ruby", Platform: "linux", Desc: "Base64 via ruby system()", Fn: func(cmd string) string {
		return Wrap(cmd, Base64Enc, RubyDec)
	}})

	// Windows encoders
	sdk.RegisterEncoder(&sdk.Encoder{Name: "cmd/powershell", Platform: "windows", Desc: "UTF-16LE base64 via powershell -enc", Fn: Base64PowerShell})
}
