package sdk

// HTTPServerFunc starts an HTTP server and returns its URL + stop func.
type HTTPServerFunc func(params Params, mux *ServerMux) (url string, stop func(), err error)

var httpServerFn HTTPServerFunc

// SetHTTPServerFunc registers the HTTP server implementation.
func SetHTTPServerFunc(fn HTTPServerFunc) { httpServerFn = fn }

// StartHTTPServer starts the HTTP server using the registered implementation.
func StartHTTPServer(params Params, mux *ServerMux) (string, func(), error) {
	if httpServerFn == nil {
		return "", nil, Errorf("no HTTP server registered (import pkg/httpsrv)")
	}
	return httpServerFn(params, mux)
}
