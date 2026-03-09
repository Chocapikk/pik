package sdk

import "testing"

func TestStartHTTPServerWithout(t *testing.T) {
	old := httpServerFn
	httpServerFn = nil
	defer func() { httpServerFn = old }()

	_, _, err := StartHTTPServer(NewParams(nil, nil), nil)
	if err == nil {
		t.Error("expected error without registration")
	}
}

func TestStartHTTPServerWith(t *testing.T) {
	old := httpServerFn
	defer func() { httpServerFn = old }()

	SetHTTPServerFunc(func(params Params, mux *ServerMux) (string, func(), error) {
		return "http://127.0.0.1:9999", func() {}, nil
	})

	url, stop, err := StartHTTPServer(NewParams(nil, nil), nil)
	if err != nil {
		t.Fatal(err)
	}
	if url != "http://127.0.0.1:9999" {
		t.Errorf("url = %q", url)
	}
	stop() // should not panic
}
