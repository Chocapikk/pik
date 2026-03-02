package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Chocapikk/pik/pkg/output"
	"github.com/Chocapikk/pik/pkg/toolchain"
)

func newCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "new [name]",
		Short: "Generate a new exploit boilerplate",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			name := args[0]
			if err := os.MkdirAll(name, 0755); err != nil {
				return fmt.Errorf("create directory: %w", err)
			}

			baseName := filepath.Base(name)
			mainPath := filepath.Join(name, "main.go")
			if err := toolchain.RenderToFile(mainPath, "boilerplate.go.tpl", map[string]string{
				"Name":       baseName,
				"StructName": toPascal(baseName),
			}); err != nil {
				return fmt.Errorf("render boilerplate: %w", err)
			}

			output.Success("Created %s", mainPath)
			output.Status("Next: cd %s && go mod init %s && go get github.com/Chocapikk/pik && go build .", name, name)
			return nil
		},
	}
}

func toPascal(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '-' || r == '_' || r == '.'
	})
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "")
}
