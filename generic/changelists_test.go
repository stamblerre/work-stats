package generic_test

import (
	"testing"

	"github.com/stamblerre/work-stats/generic"
)

func TestExtractCategory(t *testing.T) {
	for _, tt := range []struct {
		cl   *generic.Changelist
		want string
	}{{
		cl: &generic.Changelist{
			Repo:   "tools",
			Number: 353890,
			AffectedFiles: []string{
				"internal/lsp/cache/parse.go",
				"internal/lsp/semantic.go",
				"internal/lsp/source/completion/package.go",
				"internal/lsp/source/extract.go",
				"internal/lsp/source/format.go",
				"internal/lsp/source/format_test.go.go",
			},
		},
		want: "internal/lsp",
	}} {
		got := tt.cl.Category()
		if got != tt.want {
			t.Errorf("expected category %q, got %q", tt.want, got)
		}
	}
}
