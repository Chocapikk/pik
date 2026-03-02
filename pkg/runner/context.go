package runner

import (
	"net/url"
	"time"

	"github.com/Chocapikk/pik/sdk"
	pikhttp "github.com/Chocapikk/pik/pkg/http"
	"github.com/Chocapikk/pik/pkg/output"
	"github.com/Chocapikk/pik/pkg/payload"
	"github.com/Chocapikk/pik/pkg/text"
)

// BuildContext creates a wired sdk.Context from params and payload command.
// Shared between the console and runner.
func BuildContext(params sdk.Params, payloadCmd string) *sdk.Context {
	ctx := sdk.NewContext(params.Map(), payloadCmd)
	ctx.StatusFn = output.Status
	ctx.SuccessFn = output.Success
	ctx.ErrorFn = output.Error
	ctx.WarningFn = output.Warning
	ctx.Base64BashFn = payload.Base64Bash
	ctx.CommentFn = payload.CommentTrail
	ctx.RandTextFn = text.RandText
	ctx.SendFn = httpBridge(params)
	return ctx
}

func httpBridge(params sdk.Params) func(sdk.Request) (*sdk.Response, error) {
	run := pikhttp.FromModule(params)
	return func(req sdk.Request) (*sdk.Response, error) {
		timeout := time.Duration(req.Timeout) * time.Second
		if req.FireAndForget && timeout == 0 {
			timeout = 3 * time.Second
		}

		resp, err := run.Send(pikhttp.Request{
			Method:      req.Method,
			Path:        req.Path,
			Query:       url.Values(req.Query),
			Form:        url.Values(req.Form),
			Body:        req.BodyReader(),
			ContentType: req.ContentType,
			Headers:     req.Headers,
			Timeout:     timeout,
			NoRedirect:  req.NoRedirect,
		})

		if req.FireAndForget {
			if resp != nil && resp.Body != nil {
				resp.Body.Close()
			}
			return &sdk.Response{StatusCode: 0}, nil
		}

		if err != nil {
			return nil, err
		}
		r := &sdk.Response{StatusCode: resp.StatusCode, Body: resp.Body}
		r.SetContainsFn(resp.ContainsAny)
		return r, nil
	}
}
