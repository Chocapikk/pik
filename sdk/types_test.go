package sdk

import (
	"errors"
	"io"
	"strings"
	"testing"
)

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) { return 0, errors.New("read error") }

func TestHTTPRequestProtocol(t *testing.T) {
	req := HTTPRequest{}
	if got := req.protocol(); got != "http" {
		t.Errorf("protocol = %q", got)
	}
}

func TestHTTPRequestBodyReader(t *testing.T) {
	req := HTTPRequest{Body: "hello"}
	r := req.BodyReader()
	data, _ := io.ReadAll(r)
	if string(data) != "hello" {
		t.Errorf("BodyReader = %q", data)
	}

	empty := HTTPRequest{}
	if empty.BodyReader() != nil {
		t.Error("empty body should return nil reader")
	}
}

func TestHTTPResponseHeader(t *testing.T) {
	resp := &HTTPResponse{Headers: map[string]string{
		"Content-Type": "text/html",
		"X-Custom":     "value",
	}}

	if got := resp.Header("Content-Type"); got != "text/html" {
		t.Errorf("exact = %q", got)
	}
	if got := resp.Header("content-type"); got != "text/html" {
		t.Errorf("case-insensitive = %q", got)
	}
	if got := resp.Header("Missing"); got != "" {
		t.Errorf("missing = %q", got)
	}

	nilResp := &HTTPResponse{}
	if got := nilResp.Header("Anything"); got != "" {
		t.Errorf("nil headers = %q", got)
	}
}

func TestHTTPResponseBodyBytes(t *testing.T) {
	resp := &HTTPResponse{Body: io.NopCloser(strings.NewReader("hello"))}
	data, err := resp.BodyBytes()
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello" {
		t.Errorf("BodyBytes = %q", data)
	}

	// Second call should return cached data
	data2, err := resp.BodyBytes()
	if err != nil || string(data2) != "hello" {
		t.Error("cached BodyBytes failed")
	}
}

func TestHTTPResponseBodyBytesNil(t *testing.T) {
	resp := &HTTPResponse{}
	data, err := resp.BodyBytes()
	if err != nil || data != nil {
		t.Errorf("nil body: data=%v, err=%v", data, err)
	}
}

func TestHTTPResponseBodyString(t *testing.T) {
	resp := &HTTPResponse{Body: io.NopCloser(strings.NewReader("world"))}
	s, err := resp.BodyString()
	if err != nil || s != "world" {
		t.Errorf("BodyString = %q, err=%v", s, err)
	}
}

func TestHTTPResponseJSON(t *testing.T) {
	resp := &HTTPResponse{Body: io.NopCloser(strings.NewReader(`{"key":"val"}`))}
	var result map[string]string
	if err := resp.JSON(&result); err != nil {
		t.Fatal(err)
	}
	if result["key"] != "val" {
		t.Errorf("JSON = %v", result)
	}
}

func TestHTTPResponseJSONInvalid(t *testing.T) {
	resp := &HTTPResponse{Body: io.NopCloser(strings.NewReader("not json"))}
	var result map[string]string
	if err := resp.JSON(&result); err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestHTTPResponseContains(t *testing.T) {
	resp := &HTTPResponse{Body: io.NopCloser(strings.NewReader("hello world"))}
	if !resp.Contains("world") {
		t.Error("should contain world")
	}
	if resp.Contains("missing") {
		t.Error("should not contain missing")
	}
}

func TestHTTPResponseContainsNilBody(t *testing.T) {
	resp := &HTTPResponse{}
	if resp.Contains("anything") {
		t.Error("nil body should not contain anything")
	}
}

func TestHTTPResponseContainsAny(t *testing.T) {
	resp := &HTTPResponse{Body: io.NopCloser(strings.NewReader("error 404 not found"))}
	if !resp.ContainsAny("200", "404") {
		t.Error("should match 404")
	}
	if resp.ContainsAny("500", "302") {
		t.Error("should not match")
	}
}

func TestHTTPResponseBodyStringError(t *testing.T) {
	resp := &HTTPResponse{Body: io.NopCloser(&errorReader{})}
	_, err := resp.BodyString()
	if err == nil {
		t.Error("expected error")
	}
}

func TestHTTPResponseJSONReadError(t *testing.T) {
	resp := &HTTPResponse{Body: io.NopCloser(&errorReader{})}
	var result map[string]string
	if err := resp.JSON(&result); err == nil {
		t.Error("expected error")
	}
}

func TestHTTPResponseContainsReadError(t *testing.T) {
	resp := &HTTPResponse{Body: io.NopCloser(&errorReader{})}
	if resp.Contains("anything") {
		t.Error("should return false on read error")
	}
}

func TestHTTPResponseContainsAnyReadError(t *testing.T) {
	resp := &HTTPResponse{Body: io.NopCloser(&errorReader{})}
	if resp.ContainsAny("a", "b") {
		t.Error("should return false on read error")
	}
}

func TestHTTPResponseContainsAnyCustomFn(t *testing.T) {
	resp := &HTTPResponse{}
	resp.SetContainsFn(func(substrs ...string) bool {
		for _, s := range substrs {
			if s == "magic" {
				return true
			}
		}
		return false
	})
	if !resp.ContainsAny("magic") {
		t.Error("custom fn should match magic")
	}
	if resp.ContainsAny("other") {
		t.Error("custom fn should not match other")
	}
}
