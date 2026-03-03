package http

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// HTML parses the response body as HTML and returns a goquery Document.
func (r *Response) HTML() (*goquery.Document, error) {
	data, err := r.BodyBytes()
	if err != nil {
		return nil, err
	}
	return goquery.NewDocumentFromReader(strings.NewReader(string(data)))
}
