//go:build darwin || linux

package markdown

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestNonRegularReadHelper(t *testing.T) {
	path := os.Getenv("TDX_FIFO_READ_HELPER")
	if path == "" {
		return
	}
	if _, err := ReadFile(path); err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("unexpected FIFO load result: %v", err)
	}
}
func TestFIFOReadFailsWithoutWaitingForWriter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.md")
	if err := unix.Mkfifo(path, 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestNonRegularReadHelper$")
	cmd.Env = append(os.Environ(), "TDX_FIFO_READ_HELPER="+path)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("FIFO load failed or blocked (%v): %s", err, out)
	}
}
