package http

import (
	"encoding/json"
	"io"
	nethttp "net/http"
	"strings"
)

// Response wraps a standard http.Response with convenience methods
// for body parsing (HTML, JSON, raw text).
type Response struct {
	*nethttp.Response

	body     []byte
	bodyRead bool
}

// WrapResponse wraps a standard *http.Response into a *Response.
func WrapResponse(resp *nethttp.Response) *Response {
	return &Response{Response: resp}
}

// BodyBytes reads the full response body and caches it.
func (r *Response) BodyBytes() ([]byte, error) {
	if r.bodyRead {
		return r.body, nil
	}
	defer r.Body.Close()
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	r.body = data
	r.bodyRead = true
	return data, nil
}

// BodyString returns the response body as a string.
func (r *Response) BodyString() (string, error) {
	data, err := r.BodyBytes()
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Contains checks whether the response body contains the given substring.
func (r *Response) Contains(substr string) bool {
	data, err := r.BodyBytes()
	if err != nil {
		return false
	}
	return strings.Contains(string(data), substr)
}

// ContainsAll checks whether the response body contains all given substrings.
func (r *Response) ContainsAll(substrs ...string) bool {
	data, err := r.BodyBytes()
	if err != nil {
		return false
	}
	body := string(data)
	for _, substr := range substrs {
		if !strings.Contains(body, substr) {
			return false
		}
	}
	return true
}

// ContainsAny checks whether the response body contains at least one of the given substrings.
func (r *Response) ContainsAny(substrs ...string) bool {
	data, err := r.BodyBytes()
	if err != nil {
		return false
	}
	body := string(data)
	for _, substr := range substrs {
		if strings.Contains(body, substr) {
			return true
		}
	}
	return false
}

// JSON unmarshals the response body into the given target.
func (r *Response) JSON(target any) error {
	data, err := r.BodyBytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}
