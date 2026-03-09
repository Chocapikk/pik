package xmlutil

import "testing"

func TestFind(t *testing.T) {
	xml := `<root><item>first</item><item>second</item></root>`
	results := Find(xml, "//item")
	if len(results) != 2 {
		t.Fatalf("Find len = %d, want 2", len(results))
	}
	if results[0] != "first" || results[1] != "second" {
		t.Errorf("Find = %v", results)
	}
}

func TestFindNested(t *testing.T) {
	xml := `<root><a><b>deep</b></a></root>`
	results := Find(xml, "//a/b")
	if len(results) != 1 || results[0] != "deep" {
		t.Errorf("Find nested = %v", results)
	}
}

func TestFindAttribute(t *testing.T) {
	xml := `<root><item id="1">one</item><item id="2">two</item></root>`
	results := Find(xml, `//item[@id="2"]`)
	if len(results) != 1 || results[0] != "two" {
		t.Errorf("Find attribute = %v", results)
	}
}

func TestFindNoMatch(t *testing.T) {
	xml := `<root><item>value</item></root>`
	results := Find(xml, "//missing")
	if len(results) != 0 {
		t.Errorf("Find no match = %v", results)
	}
}

func TestFindInvalidXML(t *testing.T) {
	results := Find("not xml at all <><>", "//item")
	if results != nil {
		t.Errorf("Find invalid XML = %v", results)
	}
}

func TestFindEmptyNodes(t *testing.T) {
	xml := `<root><item></item><item>value</item></root>`
	results := Find(xml, "//item")
	// Empty nodes are skipped
	if len(results) != 1 || results[0] != "value" {
		t.Errorf("Find empty nodes = %v", results)
	}
}
