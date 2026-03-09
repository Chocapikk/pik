package sdk

import (
	"io"
	"testing"
)

func TestXMLFindWithoutRegistration(t *testing.T) {
	old := xmlFindFn
	xmlFindFn = nil
	defer func() { xmlFindFn = old }()

	resp := &HTTPResponse{body: []byte("<root/>"), bodyRead: true}
	results := resp.XMLFind("//item")
	if results != nil {
		t.Errorf("XMLFind without registration should return nil, got %v", results)
	}
}

func TestXMLFindBodyError(t *testing.T) {
	old := xmlFindFn
	defer func() { xmlFindFn = old }()

	SetXMLFind(func(body, xpath string) []string {
		return []string{"should not reach"}
	})

	// Body that errors on read - reuses errorReader from types_test.go
	resp := &HTTPResponse{Body: io.NopCloser(&errorReader{})}
	results := resp.XMLFind("//item")
	if results != nil {
		t.Errorf("XMLFind with body error should return nil, got %v", results)
	}
}

func TestXMLFindWithRegistration(t *testing.T) {
	old := xmlFindFn
	defer func() { xmlFindFn = old }()

	SetXMLFind(func(body, xpath string) []string {
		return []string{"found"}
	})

	resp := &HTTPResponse{body: []byte("<root><item>found</item></root>"), bodyRead: true}
	results := resp.XMLFind("//item")
	if len(results) != 1 || results[0] != "found" {
		t.Errorf("XMLFind = %v", results)
	}
}
