package markdown

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/gofrs/flock"
)

type failingReplacement struct {
	name  string
	stage string
}

func (f *failingReplacement) Name() string { return f.name }
func (f *failingReplacement) Chmod(os.FileMode) error {
	if f.stage == "chmod" {
		return errors.New("injected chmod failure")
	}
	return nil
}
func (f *failingReplacement) WriteString(content string) (int, error) {
	if f.stage == "write" {
		return 0, errors.New("injected write failure")
	}
	return len(content), nil
}
func (f *failingReplacement) Sync() error {
	if f.stage == "sync" {
		return errors.New("injected sync failure")
	}
	return nil
}
func (f *failingReplacement) Close() error {
	if f.stage == "close" {
		f.stage = ""
		return errors.New("injected close failure")
	}
	return nil
}

func TestSaveHelperProcess(t *testing.T) {
	if os.Getenv("TDX_SAVE_HELPER") != "1" {
		return
	}
	path := os.Getenv("TDX_SAVE_PATH")
	index, err := strconv.Atoi(os.Getenv("TDX_SAVE_INDEX"))
	if err != nil {
		t.Fatal(err)
	}
	fm, err := ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	todo := fm.Todos[index]
	if err := fm.UpdateTodoItem(index, todo.Text, true); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(os.Getenv("TDX_SAVE_READY"), []byte("ready"), 0o600); err != nil {
		t.Fatal(err)
	}
	gate := os.Getenv("TDX_SAVE_GATE")
	deadline := time.Now().Add(10 * time.Second)
	for {
		if _, err := os.Stat(gate); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for save gate")
		}
		time.Sleep(5 * time.Millisecond)
	}

	result := "ok"
	if err := WriteFile(path, fm); errors.Is(err, ErrFileChanged) {
		result = "conflict"
	} else if err != nil {
		result = "error: " + err.Error()
	}
	if err := os.WriteFile(os.Getenv("TDX_SAVE_RESULT"), []byte(result), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestSaveCrashHelperProcess(t *testing.T) {
	if os.Getenv("TDX_SAVE_CRASH_HELPER") != "1" {
		return
	}
	fm, err := ReadFile(os.Getenv("TDX_SAVE_PATH"))
	if err != nil {
		t.Fatal(err)
	}
	fm.AddTodoItem("new", false)
	wanted := os.Getenv("TDX_SAVE_CRASH_STAGE")
	saveStageHook = func(stage string) {
		if stage != wanted {
			return
		}
		if err := os.WriteFile(os.Getenv("TDX_SAVE_READY"), []byte(stage), 0o600); err != nil {
			panic(err)
		}
		for {
			time.Sleep(time.Hour)
		}
	}
	var saveErr error
	if wanted == "captured-before" {
		saveErr = WriteFileUnchecked(os.Getenv("TDX_SAVE_PATH"), fm)
	} else {
		saveErr = WriteFile(os.Getenv("TDX_SAVE_PATH"), fm)
	}
	if saveErr != nil {
		t.Fatal(saveErr)
	}
}

func isolateSaveLocks(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
}

func TestWriteFileDetectsTimestampPreservingEdit(t *testing.T) {
	isolateSaveLocks(t)
	path := filepath.Join(t.TempDir(), "todo.md")
	original := "# Todos\n\n- [ ] alpha\n"
	external := "# Todos\n\n- [ ] bravo\n"
	if len(original) != len(external) {
		t.Fatal("test content must have the same size")
	}
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	fm, err := ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(external), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}

	err = WriteFile(path, fm)
	if !errors.Is(err, ErrFileChanged) {
		t.Fatalf("WriteFile() error = %v, want ErrFileChanged", err)
	}
	var conflict *ConflictError
	if !errors.As(err, &conflict) || conflict.DiskContent != external {
		t.Fatalf("conflict = %#v, want external content", conflict)
	}
	got, _ := os.ReadFile(path)
	if string(got) != external {
		t.Fatalf("disk content = %q, want external content", got)
	}
}

func TestWriteFileDetectsAtomicExternalReplacement(t *testing.T) {
	isolateSaveLocks(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "todo.md")
	original := "# Todos\n\n- [ ] original\n"
	external := "# Todos\n\n- [ ] replaced externally\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	fm, err := ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	replacement, err := os.CreateTemp(dir, ".external-*.tmp")
	if err != nil {
		t.Fatal(err)
	}
	replacementPath := replacement.Name()
	if _, err := replacement.WriteString(external); err != nil {
		_ = replacement.Close()
		t.Fatal(err)
	}
	if err := replacement.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(replacementPath, path); err != nil {
		t.Fatal(err)
	}

	err = WriteFile(path, fm)
	if !errors.Is(err, ErrFileChanged) {
		t.Fatalf("WriteFile() error = %v, want ErrFileChanged", err)
	}
	var conflict *ConflictError
	if !errors.As(err, &conflict) || conflict.DiskContent != external {
		t.Fatalf("conflict = %#v, want replacement content", conflict)
	}
	got, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(got) != external {
		t.Fatalf("disk content = %q, want replacement content", got)
	}
}

func TestConditionalWritesRequireLoadedRevision(t *testing.T) {
	isolateSaveLocks(t)
	path := filepath.Join(t.TempDir(), "todo.md")
	original := "# authoritative\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	fm := ParseMarkdown("# local\n")
	fm.FilePath = path

	if err := WriteFile(path, fm); !errors.Is(err, ErrRevisionUnknown) {
		t.Fatalf("WriteFile() error = %v, want ErrRevisionUnknown", err)
	}
	if err := WriteContent(path, "# exact local\n", fm); !errors.Is(err, ErrRevisionUnknown) {
		t.Fatalf("WriteContent() error = %v, want ErrRevisionUnknown", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != original {
		t.Fatalf("conditional write changed disk to %q", got)
	}
}

func TestCloneRetainsLoadedRevisionForConditionalSave(t *testing.T) {
	isolateSaveLocks(t)
	path := filepath.Join(t.TempDir(), "todo.md")
	if err := os.WriteFile(path, []byte("# Todos\n\n- [ ] task\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	fm, err := ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	clone := fm.Clone()
	if err := clone.UpdateTodoItem(0, "task", true); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(path, clone); err != nil {
		t.Fatalf("WriteFile(clone) error = %v", err)
	}
	if clone.FilePath != fm.FilePath || !reflect.DeepEqual(clone.Metadata, fm.Metadata) {
		t.Fatal("Clone() did not retain file identity and metadata")
	}
}

func TestNewFileKeepsSecureTemporaryMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("portable Windows mode bits do not express ACL security")
	}
	isolateSaveLocks(t)
	path := filepath.Join(t.TempDir(), "todo.md")
	fm, err := ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(path, fm); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got&0o077 != 0 {
		t.Fatalf("new file mode = %o, want no group or other permissions", got)
	}
}

func TestWriteFileConflictsWhenAbsentPathCreated(t *testing.T) {
	isolateSaveLocks(t)
	path := filepath.Join(t.TempDir(), "todo.md")
	fm, err := ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	external := "# created elsewhere\n"
	if err := os.WriteFile(path, []byte(external), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(path, fm); !errors.Is(err, ErrFileChanged) {
		t.Fatalf("WriteFile() error = %v, want ErrFileChanged", err)
	}
	got, _ := os.ReadFile(path)
	if string(got) != external {
		t.Fatalf("concurrently created file was overwritten: %q", got)
	}
}

func TestConcurrentWritersOnlyOneCommits(t *testing.T) {
	isolateSaveLocks(t)
	path := filepath.Join(t.TempDir(), "todo.md")
	if err := os.WriteFile(path, []byte("# Todos\n\n- [ ] one\n- [ ] two\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	first, err := ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	_ = first.UpdateTodoItem(0, "one", true)
	_ = second.UpdateTodoItem(1, "two", true)

	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for _, fm := range []*FileModel{first, second} {
		wg.Add(1)
		go func(model *FileModel) {
			defer wg.Done()
			<-start
			errs <- WriteFile(path, model)
		}(fm)
	}
	close(start)
	wg.Wait()
	close(errs)

	var success, conflict int
	for err := range errs {
		switch {
		case err == nil:
			success++
		case errors.Is(err, ErrFileChanged):
			conflict++
		default:
			t.Fatalf("unexpected concurrent save error: %v", err)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatalf("success=%d conflict=%d, want 1 each", success, conflict)
	}
	got, _ := os.ReadFile(path)
	firstCandidate := SerializeMarkdown(first)
	secondCandidate := SerializeMarkdown(second)
	if string(got) != firstCandidate && string(got) != secondCandidate {
		t.Fatalf("final file is not a complete candidate: %q", got)
	}
}

func TestIndependentProcessesOnlyOneCommits(t *testing.T) {
	isolateSaveLocks(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "todo.md")
	initial := "# Todos\n\n- [ ] one\n- [ ] two\n"
	if err := os.WriteFile(path, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}
	gate := filepath.Join(dir, "gate")
	type child struct {
		cmd    *exec.Cmd
		output *bytes.Buffer
		ready  string
		result string
	}
	children := make([]child, 2)
	for i := range children {
		children[i].ready = filepath.Join(dir, fmt.Sprintf("ready-%d", i))
		children[i].result = filepath.Join(dir, fmt.Sprintf("result-%d", i))
		children[i].cmd = exec.Command(os.Args[0], "-test.run=^TestSaveHelperProcess$")
		children[i].output = &bytes.Buffer{}
		children[i].cmd.Stdout = children[i].output
		children[i].cmd.Stderr = children[i].output
		children[i].cmd.Env = append(os.Environ(),
			"TDX_SAVE_HELPER=1",
			"TDX_SAVE_PATH="+path,
			"TDX_SAVE_INDEX="+strconv.Itoa(i),
			"TDX_SAVE_READY="+children[i].ready,
			"TDX_SAVE_RESULT="+children[i].result,
			"TDX_SAVE_GATE="+gate,
		)
		if err := children[i].cmd.Start(); err != nil {
			t.Fatal(err)
		}
	}

	deadline := time.Now().Add(10 * time.Second)
	for _, child := range children {
		for {
			if _, err := os.Stat(child.ready); err == nil {
				break
			}
			if time.Now().After(deadline) {
				t.Fatal("timed out waiting for helper readiness")
			}
			time.Sleep(5 * time.Millisecond)
		}
	}
	if err := os.WriteFile(gate, []byte("go"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, child := range children {
		if err := child.cmd.Wait(); err != nil {
			t.Fatalf("save helper failed: %v\n%s", err, child.output.String())
		}
	}

	counts := map[string]int{}
	for _, child := range children {
		result, err := os.ReadFile(child.result)
		if err != nil {
			t.Fatal(err)
		}
		counts[string(result)]++
	}
	if counts["ok"] != 1 || counts["conflict"] != 1 {
		t.Fatalf("process results = %v, want one success and one conflict", counts)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "# Todos\n\n- [x] one\n- [ ] two\n" && string(got) != "# Todos\n\n- [ ] one\n- [x] two\n" {
		t.Fatalf("final file is not one complete candidate: %q", got)
	}
}

func TestWritePreservesSymlinkAndPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation and Unix permission assertions require elevated setup on Windows")
	}
	isolateSaveLocks(t)
	dir := t.TempDir()
	target := filepath.Join(dir, "target.md")
	link := filepath.Join(dir, "todo.md")
	if err := os.WriteFile(target, []byte("# Todos\n\n- [ ] task\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(target, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	fm, err := ReadFile(link)
	if err != nil {
		t.Fatal(err)
	}
	_ = fm.UpdateTodoItem(0, "task", true)
	if err := WriteFile(link, fm); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("save replaced the symlink")
	}
	targetInfo, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if targetInfo.Mode().Perm() != 0o600 {
		t.Fatalf("target mode = %o, want 600", targetInfo.Mode().Perm())
	}
	got, _ := os.ReadFile(target)
	if string(got) != SerializeMarkdown(fm) {
		t.Fatalf("target content = %q, want saved content", got)
	}
}

func TestConflictCleansPreparedReplacement(t *testing.T) {
	isolateSaveLocks(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "todo.md")
	if err := os.WriteFile(path, []byte("# Todos\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	fm, err := ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("# external\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(path, fm); !errors.Is(err, ErrFileChanged) {
		t.Fatalf("WriteFile() error = %v, want conflict", err)
	}
	matches, err := filepath.Glob(filepath.Join(dir, ".todo.md.tdx-*.tmp"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary files remain after conflict: %v", matches)
	}
}

func TestReplaceFailureLeavesOriginalAndCleansTemp(t *testing.T) {
	isolateSaveLocks(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "todo.md")
	original := "# original\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	fm, err := ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	fm.AddTodoItem("new", false)
	originalRename := renameFile
	renameFile = func(string, string) error { return errors.New("injected replace failure") }
	t.Cleanup(func() { renameFile = originalRename })

	if err := WriteFile(path, fm); err == nil {
		t.Fatal("WriteFile() succeeded with injected replace failure")
	}
	got, _ := os.ReadFile(path)
	if string(got) != original {
		t.Fatalf("original changed after replace failure: %q", got)
	}
	matches, _ := filepath.Glob(filepath.Join(dir, ".todo.md.tdx-*.tmp"))
	if len(matches) != 0 {
		t.Fatalf("temporary files remain after replace failure: %v", matches)
	}
}

func TestPrepareReplacementStageFailures(t *testing.T) {
	target := filepath.Join(t.TempDir(), "todo.md")
	if err := os.WriteFile(target, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	originalCreate := createTempFile
	originalStat := statTarget
	t.Cleanup(func() {
		createTempFile = originalCreate
		statTarget = originalStat
	})

	t.Run("stat", func(t *testing.T) {
		statTarget = func(string) (os.FileInfo, error) { return nil, errors.New("injected stat failure") }
		if _, err := prepareReplacement(target, "content"); err == nil {
			t.Fatal("prepareReplacement() succeeded with stat failure")
		}
		statTarget = originalStat
	})
	t.Run("create", func(t *testing.T) {
		createTempFile = func(string, string) (replacementFile, error) {
			return nil, errors.New("injected create failure")
		}
		if _, err := prepareReplacement(target, "content"); err == nil {
			t.Fatal("prepareReplacement() succeeded with create failure")
		}
		createTempFile = originalCreate
	})
	for _, stage := range []string{"chmod", "write", "sync", "close"} {
		t.Run(stage, func(t *testing.T) {
			createTempFile = func(string, string) (replacementFile, error) {
				return &failingReplacement{name: filepath.Join(t.TempDir(), "fake.tmp"), stage: stage}, nil
			}
			if _, err := prepareReplacement(target, "content"); err == nil {
				t.Fatalf("prepareReplacement() succeeded with %s failure", stage)
			}
			createTempFile = originalCreate
		})
	}
}

func TestRevisionReadFailureLeavesOriginalAndCleansTemp(t *testing.T) {
	isolateSaveLocks(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "todo.md")
	original := "# original\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	fm, err := ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	originalRead := readCurrentRevision
	readCurrentRevision = func(string) (fileRevision, string, time.Time, error) {
		return fileRevision{}, "", time.Time{}, errors.New("injected revision read failure")
	}
	t.Cleanup(func() { readCurrentRevision = originalRead })
	if err := WriteFile(path, fm); err == nil {
		t.Fatal("WriteFile() succeeded with revision read failure")
	}
	got, _ := os.ReadFile(path)
	if string(got) != original {
		t.Fatalf("revision read failure changed original: %q", got)
	}
	matches, _ := filepath.Glob(filepath.Join(dir, ".todo.md.tdx-*.tmp"))
	if len(matches) != 0 {
		t.Fatalf("temporary files remain after revision failure: %v", matches)
	}
}

func TestDirectorySyncFailureIsPostCommit(t *testing.T) {
	isolateSaveLocks(t)
	path := filepath.Join(t.TempDir(), "todo.md")
	fm, err := ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	fm.AddTodoItem("saved", false)
	originalSync := syncParentDirectory
	syncParentDirectory = func(string) error { return errors.New("injected directory sync failure") }
	t.Cleanup(func() { syncParentDirectory = originalSync })
	err = WriteFile(path, fm)
	var postCommit *PostCommitError
	if !errors.As(err, &postCommit) {
		t.Fatalf("WriteFile() error = %v, want PostCommitError", err)
	}
	got, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(got) != SerializeMarkdown(fm) {
		t.Fatalf("directory sync failure did not retain committed content: %q", got)
	}
}

func TestForcePreimageCaptureFailureLeavesOriginal(t *testing.T) {
	store := Store{}
	isolateSaveLocks(t)
	path := filepath.Join(t.TempDir(), "todo.md")
	original := "# external\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	originalReadHook := store.OnRead
	store.OnRead = func(string, string) error { return errors.New("capture failed") }
	t.Cleanup(func() { store.OnRead = originalReadHook })
	if err := store.WriteContentUnchecked(path, "# local\n"); err == nil {
		t.Fatal("store.WriteContentUnchecked() succeeded despite failed preimage capture")
	}
	got, _ := os.ReadFile(path)
	if string(got) != original {
		t.Fatalf("force-save changed original after capture failure: %q", got)
	}
}

func TestCrashBoundariesNeverExposePartialTarget(t *testing.T) {
	for _, stage := range []string{"prepared", "locked", "validated", "captured-before", "replaced", "synced", "captured-after", "unlocked"} {
		t.Run(stage, func(t *testing.T) {
			isolateSaveLocks(t)
			dir := t.TempDir()
			path := filepath.Join(dir, "todo.md")
			original := "# Todos\n\n- [ ] old\n"
			if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
				t.Fatal(err)
			}
			ready := filepath.Join(dir, "ready")
			cmd := exec.Command(os.Args[0], "-test.run=^TestSaveCrashHelperProcess$")
			cmd.Env = append(os.Environ(),
				"TDX_SAVE_CRASH_HELPER=1",
				"TDX_SAVE_PATH="+path,
				"TDX_SAVE_CRASH_STAGE="+stage,
				"TDX_SAVE_READY="+ready,
			)
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			deadline := time.Now().Add(10 * time.Second)
			for {
				if _, err := os.Stat(ready); err == nil {
					break
				}
				if time.Now().After(deadline) {
					_ = cmd.Process.Kill()
					t.Fatal("timed out waiting for crash stage")
				}
				time.Sleep(5 * time.Millisecond)
			}
			if err := cmd.Process.Kill(); err != nil {
				t.Fatal(err)
			}
			_ = cmd.Wait()

			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			completeReplacement := "# Todos\n\n- [ ] old\n- [ ] new\n"
			beforeCommit := stage == "prepared" || stage == "locked" || stage == "validated" || stage == "captured-before"
			if beforeCommit && string(got) != original {
				t.Fatalf("prepared-stage crash changed target: %q", got)
			}
			if !beforeCommit && string(got) != completeReplacement {
				t.Fatalf("replaced-stage crash left incomplete target: %q", got)
			}
			recovered, err := ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			recovered.AddTodoItem("after crash", false)
			if err := WriteFile(path, recovered); err != nil {
				t.Fatalf("save after crashed lock holder failed: %v", err)
			}
		})
	}
}

func TestPostCommitHookErrorReportsCommittedSave(t *testing.T) {
	store := Store{}
	isolateSaveLocks(t)
	originalHook := store.OnWrite
	defer func() { store.OnWrite = originalHook }()
	hookErr := errors.New("version store unavailable")
	store.OnWrite = func(string, string) error { return hookErr }

	path := filepath.Join(t.TempDir(), "todo.md")
	fm, err := store.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	fm.AddTodoItem("saved", false)
	err = store.WriteFile(path, fm)
	var postCommit *PostCommitError
	if !errors.As(err, &postCommit) || !errors.Is(err, hookErr) {
		t.Fatalf("store.WriteFile() error = %v, want PostCommitError wrapping hook error", err)
	}
	modified, checkErr := fm.CheckFileModified()
	if checkErr != nil || modified {
		t.Fatalf("committed model revision not updated: modified=%v err=%v", modified, checkErr)
	}
	got, _ := os.ReadFile(path)
	if string(got) != SerializeMarkdown(fm) {
		t.Fatalf("committed content = %q, want model content", got)
	}
}

func TestForceSaveCapturesOverwrittenContent(t *testing.T) {
	store := Store{}
	isolateSaveLocks(t)
	originalReadHook := store.OnRead
	originalWriteHook := store.OnWrite
	defer func() {
		store.OnRead = originalReadHook
		store.OnWrite = originalWriteHook
	}()
	var before, after string
	store.OnRead = func(_ string, content string) error {
		before = content
		return nil
	}
	store.OnWrite = func(_ string, content string) error {
		after = content
		return nil
	}

	path := filepath.Join(t.TempDir(), "todo.md")
	external := "# external\n"
	if err := os.WriteFile(path, []byte(external), 0o644); err != nil {
		t.Fatal(err)
	}
	fm := ParseMarkdown("# local\n")
	fm.FilePath = path
	if err := store.WriteFileUnchecked(path, fm); err != nil {
		t.Fatal(err)
	}
	if before != external || after != SerializeMarkdown(fm) {
		t.Fatalf("captured before=%q after=%q", before, after)
	}
}

func TestMultipleConditionalWritesUpdateRevisionImmediately(t *testing.T) {
	isolateSaveLocks(t)
	path := filepath.Join(t.TempDir(), "todo.md")
	if err := os.WriteFile(path, []byte("# Todos\n\n- [ ] one\n- [ ] two\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	fm, err := ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	_ = fm.UpdateTodoItem(0, "one", true)
	if err := WriteFile(path, fm); err != nil {
		t.Fatal(err)
	}
	_ = fm.UpdateTodoItem(1, "two", true)
	if err := WriteFile(path, fm); err != nil {
		t.Fatal(err)
	}
	modified, err := fm.CheckFileModified()
	if err != nil || modified {
		t.Fatalf("modified=%v err=%v after consecutive writes", modified, err)
	}
}

func TestLockTimeoutIsBounded(t *testing.T) {
	if testing.Short() {
		t.Skip("lock timeout test waits for the configured bound")
	}
	isolateSaveLocks(t)
	path := filepath.Join(t.TempDir(), "todo.md")
	if err := os.WriteFile(path, []byte("# Todos\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	fm, err := ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	target, _ := resolveTarget(path)
	lockFile, err := lockPath(target)
	if err != nil {
		t.Fatal(err)
	}
	held := flock.New(lockFile)
	if err := held.Lock(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := held.Unlock(); err != nil {
			t.Errorf("Unlock() error: %v", err)
		}
	})

	started := time.Now()
	err = WriteFile(path, fm)
	if !errors.Is(err, ErrFileBusy) {
		t.Fatalf("WriteFile() error = %v, want ErrFileBusy", err)
	}
	if elapsed := time.Since(started); elapsed < lockWait || elapsed > lockWait+time.Second {
		t.Fatalf("lock wait = %v, want bounded near %v", elapsed, lockWait)
	}
}
