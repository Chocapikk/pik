package sdk

// XMLFind returns the inner text of all XML nodes matching the XPath expression.
// Requires a parser registered via SetXMLFind (see pkg/xmlutil).
func (r *HTTPResponse) XMLFind(xpath string) []string {
	if xmlFindFn == nil {
		return nil
	}
	data, err := r.BodyString()
	if err != nil {
		return nil
	}
	return xmlFindFn(data, xpath)
}

// SetXMLFind registers the XPath query implementation.
// Called by pkg/xmlutil.init() to avoid pulling xmlquery into all binaries.
func SetXMLFind(fn func(body, xpath string) []string) { xmlFindFn = fn }

var xmlFindFn func(body, xpath string) []string
