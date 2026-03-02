package sdk

import (
	"encoding/json"
	"io"
	"strings"
)

// Values is a map of string slices, used for query/form parameters.
type Values = map[string][]string

// Request describes an HTTP request from module code.
type Request struct {
	Method         string
	Path           string
	Query          Values
	Form           Values
	Body           string // raw request body
	ContentType    string
	Headers        map[string]string
	Timeout        int  // seconds
	NoRedirect     bool
	FireAndForget  bool // send request, ignore response and errors
}

// BodyReader returns the Body as an io.Reader. Used internally by the HTTP bridge.
func (r *Request) BodyReader() io.Reader {
	if r.Body == "" {
		return nil
	}
	return strings.NewReader(r.Body)
}

// Response is an HTTP response for module code.
type Response struct {
	StatusCode int
	Body       io.ReadCloser
	body       []byte
	bodyRead   bool
	containsFn func(...string) bool
}

// SetContainsFn sets the function used by ContainsAny.
func (r *Response) SetContainsFn(fn func(...string) bool) {
	r.containsFn = fn
}

// BodyBytes reads and caches the full response body.
func (r *Response) BodyBytes() ([]byte, error) {
	if r.bodyRead {
		return r.body, nil
	}
	if r.Body == nil {
		r.bodyRead = true
		return nil, nil
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

// JSON unmarshals the response body into the given target.
func (r *Response) JSON(target any) error {
	data, err := r.BodyBytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

// Contains checks if the response body contains the given substring.
func (r *Response) Contains(substr string) bool {
	data, err := r.BodyBytes()
	if err != nil {
		return false
	}
	return strings.Contains(string(data), substr)
}

// ContainsAny returns true if the response body contains any of the given substrings.
func (r *Response) ContainsAny(substrs ...string) bool {
	if r.containsFn != nil {
		return r.containsFn(substrs...)
	}
	data, err := r.BodyBytes()
	if err != nil {
		return false
	}
	body := string(data)
	for _, s := range substrs {
		if strings.Contains(body, s) {
			return true
		}
	}
	return false
}
