package payload

import (
	"strings"
	"testing"
)

func TestBindShellsContainPort(t *testing.T) {
	port := 9999
	generators := []struct {
		name string
		fn   func(int) string
	}{
		{"NetcatBind", NetcatBind},
		{"NetcatMkfifoBind", NetcatMkfifoBind},
		{"PythonBind", PythonBind},
		{"PHPBind", PHPBind},
		{"SocatBind", SocatBind},
	}
	for _, gen := range generators {
		t.Run(gen.name, func(t *testing.T) {
			result := gen.fn(port)
			if !strings.Contains(result, "9999") {
				t.Errorf("%s does not contain port %d", gen.name, port)
			}
			if result == "" {
				t.Errorf("%s returned empty", gen.name)
			}
		})
	}
}
