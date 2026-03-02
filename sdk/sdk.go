// Package sdk is the public facade for Pik module development.
// It re-exports types and functions from internal packages so that
// module authors only need a single import.
//
// Internal framework packages (pkg/runner, pkg/console, etc.) import
// pkg/core directly to avoid import cycles.
package sdk

import (
	"github.com/Chocapikk/pik/pkg/c2/shell"
	"github.com/Chocapikk/pik/pkg/core"
	"github.com/Chocapikk/pik/pkg/log"
	"github.com/Chocapikk/pik/pkg/output"
	"github.com/Chocapikk/pik/pkg/payload"
	"github.com/Chocapikk/pik/pkg/text"

	pikhttp "github.com/Chocapikk/pik/pkg/http"
)

// --- Core types ---

type (
	Exploit     = core.Exploit
	Pik         = core.Pik
	Checker     = core.Checker
	CmdStager   = core.CmdStager
	Info        = core.Info
	Notes       = core.Notes
	Stance      = core.Stance
	Context     = core.Context
	Request     = core.Request
	Response    = core.Response
	Values      = core.Values
	CheckResult = core.CheckResult
	CheckCode   = core.CheckCode
	Reliability = core.Reliability
	Reference   = core.Reference
	RefType     = core.RefType
	Query       = core.Query
	Target      = core.Target
	Option      = core.Option
	OptionType  = core.OptionType
	Params      = core.Params
	AuthorRank  = core.AuthorRank
)

// --- Reliability constants ---

const (
	Unstable     = core.Unstable
	Unlikely     = core.Unlikely
	Difficult    = core.Difficult
	Typical      = core.Typical
	Reliable     = core.Reliable
	VeryReliable = core.VeryReliable
	Certain      = core.Certain
)

// --- CheckCode constants ---

const (
	CheckUnknown    = core.CheckUnknown
	CheckSafe       = core.CheckSafe
	CheckDetected   = core.CheckDetected
	CheckAppears    = core.CheckAppears
	CheckVulnerable = core.CheckVulnerable
)

// --- Stance constants ---

const (
	Aggressive = core.Aggressive
	Passive    = core.Passive
)

// --- Notes tags ---

const (
	CrashSafe         = core.CrashSafe
	CrashUnsafe       = core.CrashUnsafe
	ServiceRestart    = core.ServiceRestart
	ArtifactsOnDisk   = core.ArtifactsOnDisk
	IOCInLogs         = core.IOCInLogs
	ConfigChanges     = core.ConfigChanges
	RepeatableSession = core.RepeatableSession
	AccountLockout    = core.AccountLockout
)

// --- Option types ---

const (
	TypeString  = core.TypeString
	TypeInt     = core.TypeInt
	TypeBool    = core.TypeBool
	TypePort    = core.TypePort
	TypePath    = core.TypePath
	TypeAddress = core.TypeAddress
	TypeEnum    = core.TypeEnum
)

// --- Check result helpers ---

var (
	CheckOK   = core.CheckOK
	CheckFail = core.CheckFail
	CheckErr  = core.CheckErr
)

// --- RefType constants ---

const (
	RefCVE         = core.RefCVE
	RefGHSA        = core.RefGHSA
	RefEDB         = core.RefEDB
	RefURL         = core.RefURL
	RefPacketstorm = core.RefPacketstorm
)

// --- Core functions ---

var (
	NewContext     = core.NewContext
	NewParams      = core.NewParams
	// Register is defined as a function below (not a var alias)
	// because it needs to adjust runtime.Caller depth.
	Get      = core.Get
	List     = core.List
	Names    = core.Names
	NameOf   = core.NameOf
	Rankings = core.Rankings
	CanCheck       = core.CanCheck
	HasOption      = core.HasOption
	ResolveOptions = core.ResolveOptions
	RegisterEnricher = core.RegisterEnricher
)

// Register adds an exploit to the global registry.
// Wraps core.Register with adjusted caller depth for the SDK indirection.
func Register(mod core.Exploit) { core.RegisterFrom(mod, 3) }

// --- Reference constructors ---

var (
	CVE         = core.CVE
	GHSA        = core.GHSA
	EDB         = core.EDB
	URL         = core.URL
	Packetstorm = core.Packetstorm
)

// --- Query constructors ---

var (
	Shodan  = core.Shodan
	FOFA    = core.FOFA
	ZoomEye = core.ZoomEye
	LeakIX  = core.LeakIX
	Google  = core.Google
	Censys  = core.Censys
	Hunter  = core.Hunter
)

// --- Target constructors ---

var (
	TargetLinux   = core.TargetLinux
	TargetWindows = core.TargetWindows
)

// --- Option constructors ---

var (
	OptTargetURI = core.OptTargetURI
	OptString    = core.OptString
	OptRequired  = core.OptRequired
	OptInt       = core.OptInt
	OptPort      = core.OptPort
	OptBool      = core.OptBool
	OptEnum      = core.OptEnum
	OptAddress   = core.OptAddress
)

// --- Convenience functions ---

var (
	Sprintf = core.Sprintf
	Errorf  = core.Errorf
	Dedent  = core.Dedent
)

// --- Logging (from pkg/log) ---

var (
	LogStatus  = log.Status
	LogSuccess = log.Success
	LogError   = log.Error
	LogWarning = log.Warning
	LogVerbose = log.Verbose
	LogDebug   = log.Debug
)

// --- Output (from pkg/output) ---

var (
	Banner  = output.Banner
	Print   = output.Print
	Println = output.Println
	Spinner = output.Spinner
)

// --- Payload helpers (from pkg/payload) ---

var (
	Base64Bash       = payload.Base64Bash
	Base64BashC      = payload.Base64BashC
	Base64Python     = payload.Base64Python
	Base64Perl       = payload.Base64Perl
	Base64PowerShell = payload.Base64PowerShell
	HexBash          = payload.HexBash
	CommentTrail     = payload.CommentTrail
	BackgroundExec   = payload.BackgroundExec
	NohupExec        = payload.NohupExec
	SemicolonChain   = payload.SemicolonChain
	PipeChain        = payload.PipeChain
	URLEncodeStr     = payload.URLEncodeStr
	DoubleURLEncodeStr = payload.DoubleURLEncodeStr
)

// Reverse shell generators
var (
	BashPayload        = payload.Bash
	BashMinPayload     = payload.BashMin
	BashFDPayload      = payload.BashFD
	PythonPayload      = payload.Python
	PerlPayload        = payload.Perl
	RubyPayload        = payload.Ruby
	PHPPayload         = payload.PHP
	NetcatPayload      = payload.Netcat
	NetcatMkfifo       = payload.NetcatMkfifo
	PowerShellPayload  = payload.PowerShell
	SocatPayload       = payload.Socat
	NodeJSPayload      = payload.NodeJS
	AwkPayload         = payload.Awk
)

// Stager generators
var (
	CurlStager       = payload.Curl
	WgetStager       = payload.Wget
	CurlPipe         = payload.CurlPipe
	WgetPipe         = payload.WgetPipe
	PHPDownload      = payload.PHPDownload
	PerlDownload     = payload.PerlDownload
	PythonDownload   = payload.PythonDownload
	PowerShellDownload = payload.PowerShellDownload
	PowerShellIEX    = payload.PowerShellIEX
	Certutil         = payload.Certutil
	Bitsadmin        = payload.Bitsadmin
	Mshta            = payload.Mshta
)

// --- Text helpers (from pkg/text) ---

var (
	RandText     = text.RandText
	RandAlpha    = text.RandAlpha
	RandAlphaNum = text.RandAlphaNum
	RandNumeric  = text.RandNumeric
	RandHex      = text.RandHex
	RandBytes    = text.RandBytes
	RandInt      = text.RandInt
	RandBool     = text.RandBool
	RandElement  = text.RandElement
	RandUserAgent = text.RandUserAgent
)

// --- HTTP types (from pkg/http) ---

type (
	HTTPRun      = pikhttp.Run
	HTTPSession  = pikhttp.Session
	HTTPRequest  = pikhttp.Request
	HTTPResponse = pikhttp.Response
	HTTPOption   = pikhttp.Option
)

var (
	HTTPFromModule = pikhttp.FromModule
	HTTPNewSession = pikhttp.NewSession
	HTTPNewRun     = pikhttp.NewRun
	HTTPWithPool   = pikhttp.WithPool
	NormalizeURI   = pikhttp.NormalizeURI
)

// --- Shell listener (from pkg/c2/shell) ---

var NewListener = shell.New
