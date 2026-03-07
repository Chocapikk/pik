// Package xmlutil provides XPath queries on XML strings.
// Import with blank identifier to register the parser:
//
//	import _ "github.com/Chocapikk/pik/pkg/xmlutil"
package xmlutil

import (
	"strings"

	"github.com/Chocapikk/pik/sdk"
	"github.com/antchfx/xmlquery"
)

func init() { sdk.SetXMLFind(Find) }

// Find returns the inner text of all XML nodes matching the XPath expression.
func Find(body, xpath string) []string {
	doc, err := xmlquery.Parse(strings.NewReader(body))
	if err != nil {
		return nil
	}
	nodes := xmlquery.Find(doc, xpath)
	result := make([]string, 0, len(nodes))
	for _, n := range nodes {
		if text := n.InnerText(); text != "" {
			result = append(result, text)
		}
	}
	return result
}
