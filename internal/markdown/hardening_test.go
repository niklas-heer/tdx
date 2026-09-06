package markdown

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

type shortReplacement struct{ *os.File }

func (f shortReplacement) WriteString(s string) (int, error) { return f.File.WriteString(s[:len(s)/2]) }
func TestShortWriteRejectsAndRemovesActualTemporaryFile(t *testing.T) {
	isolateSaveLocks(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.md")
	original := "- [ ] Keep me\n"
	if err := os.WriteFile(path, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	model, err := ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	model.AddTodoItem("new task", false)
	create := createTempFile
	createTempFile = func(dir, pattern string) (replacementFile, error) {
		f, err := os.CreateTemp(dir, pattern)
		return shortReplacement{f}, err
	}
	t.Cleanup(func() { createTempFile = create })
	if err := WriteFile(path, model); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("want short write, got %v", err)
	}
	actual, err := os.ReadFile(path)
	if err != nil || string(actual) != original {
		t.Fatalf("target changed: %q %v", actual, err)
	}
	matches, err := filepath.Glob(filepath.Join(dir, ".*.tdx-*.tmp"))
	if err != nil || len(matches) > 0 {
		t.Fatalf("temporary files remain: %v %v", matches, err)
	}
	createTempFile = create
	if err := WriteFile(path, model); err != nil {
		t.Fatalf("could not retry rejected save: %v", err)
	}
}
func TestLogicalTargetRetargetDuringPreparation(t *testing.T) {
	for _, force := range []bool{false, true} {
		t.Run(map[bool]string{false: "conditional", true: "force"}[force], func(t *testing.T) {
			isolateSaveLocks(t)
			dir := t.TempDir()
			first, second, link := filepath.Join(dir, "first.md"), filepath.Join(dir, "second.md"), filepath.Join(dir, "tasks.md")
			for _, path := range []string{first, second} {
				if err := os.WriteFile(path, []byte("- [ ] "+filepath.Base(path)+"\n"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.Symlink(first, link); err != nil {
				t.Skipf("symlinks unavailable: %v", err)
			}
			model, err := ReadFile(link)
			if err != nil {
				t.Fatal(err)
			}
			model.AddTodoItem("local", false)
			previous := saveStageHook
			saveStageHook = func(stage string) {
				if stage == "prepared" {
					if err := os.Remove(link); err != nil {
						t.Fatal(err)
					}
					if err := os.Symlink(second, link); err != nil {
						t.Fatal(err)
					}
				}
			}
			t.Cleanup(func() { saveStageHook = previous })
			if force {
				err = WriteFileUnchecked(link, model)
			} else {
				err = WriteFile(link, model)
			}
			if !errors.Is(err, ErrFileChanged) {
				t.Fatalf("retarget not rejected: %v", err)
			}
			for _, path := range []string{first, second} {
				got, err := os.ReadFile(path)
				if err != nil || string(got) != "- [ ] "+filepath.Base(path)+"\n" {
					t.Fatalf("changed target %s: %q %v", path, got, err)
				}
			}
			matches, _ := filepath.Glob(filepath.Join(dir, ".*.tdx-*.tmp"))
			if len(matches) > 0 {
				t.Fatalf("temporary files remain: %v", matches)
			}
		})
	}
}
