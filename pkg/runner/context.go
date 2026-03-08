package runner

import (
	_ "github.com/Chocapikk/pik/pkg/enricher" // register all protocol enrichers
	"github.com/Chocapikk/pik/pkg/output"
	"github.com/Chocapikk/pik/pkg/payload"
	"github.com/Chocapikk/pik/pkg/text"
	"github.com/Chocapikk/pik/sdk"
)

// BuildContext creates a wired sdk.Context from params and payload command.
// Shared between the console and runner.
func BuildContext(params sdk.Params, payloadCmd string) *sdk.Context {
	ctx := sdk.NewContext(params.Map(), payloadCmd)
	ctx.StatusFn = output.Status
	ctx.SuccessFn = output.Success
	ctx.ErrorFn = output.Error
	ctx.WarningFn = output.Warning
	ctx.CommentFn = payload.CommentTrail
	ctx.RandTextFn = text.RandText
	sdk.WireSenders(ctx, params)
	ctx.DialFn = func() (sdk.Conn, error) { return sdk.DialWith(params) }
	return ctx
}
