package http

import (
	"net/url"
	"time"

	"github.com/Chocapikk/pik/sdk"
)

func init() {
	sdk.SetPoolFactory(WithPool)
	sdk.RegisterSenderFactory("http", func(params sdk.Params) any {
		run := FromModule(params)
		return func(req sdk.HTTPRequest) (*sdk.HTTPResponse, error) {
			timeout := time.Duration(req.Timeout) * time.Second
			if req.FireAndForget && timeout == 0 {
				timeout = 3 * time.Second
			}

			resp, err := run.Send(Request{
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
				return &sdk.HTTPResponse{StatusCode: 0}, nil
			}

			if err != nil {
				return nil, err
			}

			headers := make(map[string]string)
			for k, vals := range resp.Header {
				if len(vals) > 0 {
					headers[k] = vals[0]
				}
			}

			r := &sdk.HTTPResponse{StatusCode: resp.StatusCode, Body: resp.Body, Headers: headers}
			r.SetContainsFn(resp.ContainsAny)
			return r, nil
		}
	})
}
