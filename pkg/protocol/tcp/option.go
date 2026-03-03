package tcp

import "github.com/Chocapikk/pik/sdk"

func init() {
	sdk.SetDialFactory(func(params sdk.Params) (sdk.Conn, error) {
		return FromModule(params)
	})
}
