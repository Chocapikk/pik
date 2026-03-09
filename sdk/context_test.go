package sdk

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestNewContext(t *testing.T) {
	ctx := NewContext(map[string]string{"TARGET": "10.0.0.1"}, "id")
	if ctx.Get("TARGET") != "10.0.0.1" {
		t.Errorf("Get = %q", ctx.Get("TARGET"))
	}
	if ctx.Payload() != "id" {
		t.Errorf("Payload = %q", ctx.Payload())
	}
}

func TestContextSendNoClient(t *testing.T) {
	ctx := NewContext(nil, "")
	_, err := ctx.Send(HTTPRequest{Method: "GET"})
	if err == nil {
		t.Error("expected error with no HTTP client")
	}
}

// unsupportedReq is a Sendable that returns an unregistered protocol.
type unsupportedReq struct{}

func (unsupportedReq) protocol() string { return "ftp" }

func TestContextSendUnsupportedProtocol(t *testing.T) {
	ctx := NewContext(nil, "")
	_, err := ctx.Send(unsupportedReq{})
	if err == nil || !Contains(err.Error(), "unsupported protocol") {
		t.Errorf("err = %v", err)
	}
}

func TestContextSendHTTP(t *testing.T) {
	ctx := NewContext(nil, "")
	called := false
	ctx.RegisterSender("http", func(req HTTPRequest) (*HTTPResponse, error) {
		called = true
		if req.Method != "GET" {
			t.Errorf("Method = %q", req.Method)
		}
		return &HTTPResponse{StatusCode: 200}, nil
	})

	resp, err := ctx.Send(HTTPRequest{Method: "GET", Path: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Error("sender not called")
	}
	if resp.StatusCode != 200 {
		t.Errorf("StatusCode = %d", resp.StatusCode)
	}
}

func TestContextDialNoClient(t *testing.T) {
	ctx := NewContext(nil, "")
	_, err := ctx.Dial()
	if err == nil {
		t.Error("expected error with no TCP client")
	}
}

func TestContextDialFn(t *testing.T) {
	ctx := NewContext(nil, "")
	called := false
	ctx.DialFn = func() (Conn, error) {
		called = true
		return nil, fmt.Errorf("test")
	}
	ctx.Dial()
	if !called {
		t.Error("DialFn not called")
	}
}

func TestContextCommandsTarget(t *testing.T) {
	ctx := NewContext(nil, "")
	ctx.SetCommands([]string{"cmd1", "cmd2"})
	if len(ctx.Commands()) != 2 {
		t.Errorf("Commands = %v", ctx.Commands())
	}

	target := Target{Name: "test", Platform: "linux"}
	ctx.SetTarget(target)
	if ctx.Target().Name != "test" {
		t.Errorf("Target = %+v", ctx.Target())
	}
}

func TestContextLogging(t *testing.T) {
	ctx := NewContext(nil, "")
	// Should not panic with nil functions
	ctx.Status("test %d", 1)
	ctx.Success("test %d", 2)
	ctx.Error("test %d", 3)
	ctx.Warning("test %d", 4)

	var logged string
	ctx.StatusFn = func(f string, a ...any) { logged = fmt.Sprintf(f, a...) }
	ctx.Status("hello %s", "world")
	if logged != "hello world" {
		t.Errorf("StatusFn = %q", logged)
	}
}

func TestContextCommentTrail(t *testing.T) {
	ctx := NewContext(nil, "")
	if got := ctx.CommentTrail("cmd"); got != "cmd #" {
		t.Errorf("default CommentTrail = %q", got)
	}

	ctx.CommentFn = func(s string) string { return s + " //" }
	if got := ctx.CommentTrail("cmd"); got != "cmd //" {
		t.Errorf("custom CommentTrail = %q", got)
	}
}

func TestContextRandText(t *testing.T) {
	ctx := NewContext(nil, "")
	if got := ctx.RandText(5); got != "x" {
		t.Errorf("default RandText = %q", got)
	}

	ctx.RandTextFn = func(n int) string { return "abc" }
	if got := ctx.RandText(3); got != "abc" {
		t.Errorf("custom RandText = %q", got)
	}
}

func TestContextEncodedPayload(t *testing.T) {
	ctx := NewContext(nil, "whoami")
	if got := ctx.EncodedPayload(); got != "whoami" {
		t.Errorf("default EncodedPayload = %q", got)
	}

	ctx.EncoderFn = func(s string) string { return "encoded:" + s }
	if got := ctx.EncodedPayload(); got != "encoded:whoami" {
		t.Errorf("custom EncodedPayload = %q", got)
	}
}

func TestContextExploitURL(t *testing.T) {
	ctx := NewContext(nil, "")
	ctx.SetExploitURL("http://10.0.0.1:8080")
	if got := ctx.ExploitURL(); got != "http://10.0.0.1:8080" {
		t.Errorf("ExploitURL = %q", got)
	}
}

func TestContextMux(t *testing.T) {
	ctx := NewContext(nil, "")
	mux := ctx.Mux()
	if mux == nil {
		t.Fatal("Mux should not be nil")
	}
	if ctx.Mux() != mux {
		t.Error("Mux should return same instance")
	}
}

func TestContextElapsed(t *testing.T) {
	ctx := NewContext(nil, "")

	// Not started yet
	if got := ctx.Elapsed(false); got != 0 {
		t.Errorf("Elapsed before start = %f", got)
	}

	ctx.Elapsed(true)
	time.Sleep(50 * time.Millisecond)
	elapsed := ctx.Elapsed(false)
	if elapsed < 0.04 {
		t.Errorf("Elapsed = %f, expected >= 0.04", elapsed)
	}
}

func TestContextParams(t *testing.T) {
	ctx := NewContext(map[string]string{"TARGET": "10.0.0.1", "LPORT": "5555"}, "")
	p := ctx.Params()
	if p.Target() != "10.0.0.1" {
		t.Errorf("Params.Target = %q", p.Target())
	}
	if p.Lport() != 5555 {
		t.Errorf("Params.Lport = %d", p.Lport())
	}
}

func TestContextWaitRoutes(t *testing.T) {
	ctx := NewContext(nil, "")
	ctx.ServeRoute("/x", "text/plain", []byte("x"))
	go func() { ctx.Mux().Match("/x") }()
	err := ctx.WaitRoutes(2, "/x")
	if err != nil {
		t.Errorf("WaitRoutes = %v", err)
	}
}

func TestContextServeRoute(t *testing.T) {
	ctx := NewContext(nil, "")
	ctx.ServeRoute("/test", "text/plain", []byte("ok"))
	ct, body, ok := ctx.Mux().Match("/test")
	if !ok || ct != "text/plain" || string(body) != "ok" {
		t.Errorf("ServeRoute: ok=%v ct=%q body=%q", ok, ct, body)
	}
}

func TestContextLoggingAllPaths(t *testing.T) {
	ctx := NewContext(nil, "")
	var got string

	ctx.SuccessFn = func(f string, a ...any) { got = fmt.Sprintf(f, a...) }
	ctx.Success("ok %d", 1)
	if got != "ok 1" {
		t.Errorf("SuccessFn = %q", got)
	}

	ctx.ErrorFn = func(f string, a ...any) { got = fmt.Sprintf(f, a...) }
	ctx.Error("fail %d", 2)
	if got != "fail 2" {
		t.Errorf("ErrorFn = %q", got)
	}

	ctx.WarningFn = func(f string, a ...any) { got = fmt.Sprintf(f, a...) }
	ctx.Warning("warn %d", 3)
	if got != "warn 3" {
		t.Errorf("WarningFn = %q", got)
	}
}

func TestRegisterSenderFactory(t *testing.T) {
	old := senderFactories
	senderFactories = nil
	defer func() { senderFactories = old }()

	RegisterSenderFactory("http", func(p Params) any {
		return func(req HTTPRequest) (*HTTPResponse, error) {
			return &HTTPResponse{StatusCode: 201}, nil
		}
	})

	ctx := NewContext(nil, "")
	params := NewParams(nil, nil)
	WireSenders(ctx, params)

	resp, err := ctx.Send(HTTPRequest{Method: "POST"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 201 {
		t.Errorf("StatusCode = %d", resp.StatusCode)
	}
}

func TestWithPool(t *testing.T) {
	old := poolFactory
	defer func() { poolFactory = old }()

	// Without factory, returns same context
	bgCtx := context.Background()
	if got := WithPool(bgCtx, 10, ""); got != bgCtx {
		t.Error("WithPool without factory should return same ctx")
	}

	// With factory
	SetPoolFactory(func(ctx context.Context, threads int, proxy string) context.Context {
		return context.WithValue(ctx, "pooled", true)
	})
	pooled := WithPool(bgCtx, 10, "")
	if pooled.Value("pooled") != true {
		t.Error("WithPool should apply factory")
	}
}

func TestDialWith(t *testing.T) {
	old := dialFactory
	defer func() { dialFactory = old }()

	// Without factory
	dialFactory = nil
	_, err := DialWith(NewParams(nil, nil))
	if err == nil {
		t.Error("expected error without dial factory")
	}

	// With factory
	SetDialFactory(func(p Params) (Conn, error) {
		return nil, Errorf("mock dial")
	})
	_, err = DialWith(NewParams(nil, nil))
	if err == nil || !Contains(err.Error(), "mock dial") {
		t.Errorf("DialWith err = %v", err)
	}
}
