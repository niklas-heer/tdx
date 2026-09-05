package cmd

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/niklas-heer/tdx/internal/markdown"
)

func TestServicesDoNotShareDependencies(t *testing.T) {
	for _, name := range []string{"alpha", "beta"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "tasks.md")
			var output bytes.Buffer
			var captured []string
			service := Service{Out: &output, GreenStyle: func(s string) string { return name + ":" + s }, Store: markdown.Store{OnWrite: func(p, content string) error {
				resolved, _ := filepath.EvalSymlinks(path)
				if p != resolved {
					t.Errorf("other store wrote %s", p)
				}
				captured = append(captured, content)
				return nil
			}}}
			if err := service.AddTodo(path, name); err != nil {
				t.Fatal(err)
			}
			if len(captured) != 1 || !strings.Contains(captured[0], name) || !strings.Contains(output.String(), name+":✓") {
				t.Fatal("dependencies crossed instances")
			}
		})
	}
}
