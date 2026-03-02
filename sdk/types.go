package sdk

import "io"

// Values is a map of string slices, used for query/form parameters.
type Values = map[string][]string

// Request describes an HTTP request from module code.
type Request struct {
	Method      string
	Path        string
	Query       Values
	Form        Values
	Body        io.Reader
	ContentType string
	Headers     map[string]string
	Timeout     int // seconds
	NoRedirect  bool
}

// Response is an HTTP response for module code.
type Response struct {
	StatusCode int
	Body       io.ReadCloser
	containsFn func(...string) bool
}

// SetContainsFn sets the function used by ContainsAny.
func (r *Response) SetContainsFn(fn func(...string) bool) {
	r.containsFn = fn
}

// ContainsAny returns true if the response body contains any of the given substrings.
func (r *Response) ContainsAny(substrs ...string) bool {
	if r.containsFn != nil {
		return r.containsFn(substrs...)
	}
	return false
}
