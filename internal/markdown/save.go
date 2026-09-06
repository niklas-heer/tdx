package markdown

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
	"github.com/niklas-heer/tdx/internal/config"
	"github.com/niklas-heer/tdx/internal/saveprotocol"
)

var (
	// ErrFileChanged identifies a conditional save rejected because disk changed.
	ErrFileChanged = errors.New("file changed externally")
	// ErrFileBusy identifies a save that could not acquire the tdx lock in time.
	ErrFileBusy = saveprotocol.ErrBusy
	// ErrRevisionUnknown identifies a conditional save attempted without a loaded baseline.
	ErrRevisionUnknown = errors.New("file revision unavailable; load with ReadFile or use an unchecked write")
)

const (
	lockWait  = 2 * time.Second
	lockRetry = 10 * time.Millisecond
)

var (
	createTempFile = func(dir, pattern string) (replacementFile, error) {
		return os.CreateTemp(dir, pattern)
	}
	statTarget          = os.Stat
	readCurrentRevision = readDiskRevision
	renameFile          = replaceFile
	syncParentDirectory = syncDirectory
	saveStageHook       func(string)
)

type replacementFile interface {
	Name() string
	Chmod(fs.FileMode) error
	WriteString(string) (int, error)
	Sync() error
	Close() error
}

type fileRevision struct {
	target string
	exists bool
	hash   [sha256.Size]byte
}

func (r fileRevision) equal(other fileRevision) bool {
	return r.target == other.target && r.exists == other.exists && (!r.exists || r.hash == other.hash)
}

// ConflictError includes the authoritative disk content observed at conflict time.
type ConflictError struct {
	Path        string
	DiskContent string
}

func (e *ConflictError) Error() string { return ErrFileChanged.Error() }
func (e *ConflictError) Unwrap() error { return ErrFileChanged }

// PostCommitError reports a failure after the markdown replacement committed.
type PostCommitError struct {
	Err error
}

func (e *PostCommitError) Error() string {
	return fmt.Sprintf("file saved, but post-save processing failed: %v", e.Err)
}

func (e *PostCommitError) Unwrap() error { return e.Err }

func (fm *FileModel) setRevision(revision fileRevision, modTime time.Time) {
	fm.revision = revision
	fm.revisionKnown = true
	fm.ModTime = modTime
}

func resolveTarget(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve file path: %w", err)
	}
	absPath = filepath.Clean(absPath)

	info, err := os.Lstat(absPath)
	if err == nil && info.Mode()&os.ModeSymlink != 0 {
		resolved, resolveErr := filepath.EvalSymlinks(absPath)
		if resolveErr != nil {
			return "", fmt.Errorf("resolve symlink %q: %w", absPath, resolveErr)
		}
		return filepath.Abs(resolved)
	}
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("inspect file path: %w", err)
	}

	resolvedDir, err := filepath.EvalSymlinks(filepath.Dir(absPath))
	if err != nil {
		return "", fmt.Errorf("resolve parent directory: %w", err)
	}
	return filepath.Join(resolvedDir, filepath.Base(absPath)), nil
}

func readDiskRevision(path string) (fileRevision, string, time.Time, error) {
	target, err := resolveTarget(path)
	if err != nil {
		return fileRevision{}, "", time.Time{}, err
	}
	info, err := os.Stat(target)
	if errors.Is(err, fs.ErrNotExist) {
		return fileRevision{target: target}, "", time.Time{}, nil
	}
	if err != nil {
		return fileRevision{}, "", time.Time{}, fmt.Errorf("stat %q: %w", target, err)
	}
	if !info.Mode().IsRegular() {
		return fileRevision{}, "", time.Time{}, fmt.Errorf("%q is not a regular file", target)
	}
	// Inspect before reading: a stable FIFO/device path must not block a load.
	content, err := os.ReadFile(target)
	if errors.Is(err, fs.ErrNotExist) {
		return fileRevision{target: target}, "", time.Time{}, nil
	}
	if err != nil {
		return fileRevision{}, "", time.Time{}, fmt.Errorf("read %q: %w", target, err)
	}
	return fileRevision{
		target: target,
		exists: true,
		hash:   sha256.Sum256(content),
	}, string(content), info.ModTime(), nil
}

func lockPath(target string) (string, error) {
	dir, err := config.GetConfigDir()
	if err != nil {
		return "", fmt.Errorf("get lock directory: %w", err)
	}
	dir = filepath.Join(dir, "locks")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create lock directory: %w", err)
	}
	name := fmt.Sprintf("%x.lock", sha256.Sum256([]byte(target)))
	return filepath.Join(dir, name), nil
}

func prepareReplacement(target, content string) (name string, err error) {
	var mode fs.FileMode
	preserveMode := false
	if info, statErr := statTarget(target); statErr == nil {
		if !info.Mode().IsRegular() {
			return "", fmt.Errorf("%q is not a regular file", target)
		}
		mode = info.Mode().Perm()
		preserveMode = true
	} else if !errors.Is(statErr, fs.ErrNotExist) {
		return "", fmt.Errorf("stat target before save: %w", statErr)
	}

	tmp, err := createTempFile(filepath.Dir(target), "."+filepath.Base(target)+".tdx-*.tmp")
	if err != nil {
		return "", fmt.Errorf("create replacement: %w", err)
	}
	name = tmp.Name()
	closed := false
	defer func() {
		if !closed {
			if closeErr := tmp.Close(); closeErr != nil {
				err = errors.Join(err, fmt.Errorf("close replacement: %w", closeErr))
			}
		}
		if err != nil {
			// Error returns clear the named result; the file still owns its path.
			if removeErr := os.Remove(tmp.Name()); removeErr != nil && !errors.Is(removeErr, fs.ErrNotExist) {
				err = errors.Join(err, fmt.Errorf("remove replacement: %w", removeErr))
			}
		}
	}()

	if preserveMode {
		if err = tmp.Chmod(mode); err != nil {
			return "", fmt.Errorf("set replacement permissions: %w", err)
		}
	}
	n, writeErr := tmp.WriteString(content)
	if writeErr != nil {
		return "", fmt.Errorf("write replacement: %w", writeErr)
	}
	if n != len(content) {
		return "", fmt.Errorf("write replacement: %w", io.ErrShortWrite)
	}
	if err = tmp.Sync(); err != nil {
		return "", fmt.Errorf("sync replacement: %w", err)
	}
	if err = tmp.Close(); err != nil {
		return "", fmt.Errorf("close replacement: %w", err)
	}
	closed = true
	return name, nil
}

// nativeSave supplies effects; ordering and failure continuation live in the
// same saveprotocol.Save used by the deterministic simulator.
type nativeSave struct {
	store                       Store
	path, target, content, temp string
	expected                    *fileRevision
	model                       *FileModel
	lock                        *flock.Flock
	current                     fileRevision
	diskContent                 string
}

func (n *nativeSave) execute(phase saveprotocol.Phase) error {
	switch phase {
	case saveprotocol.Prepare:
		target, err := resolveTarget(n.path)
		if err != nil {
			return err
		}
		n.target = target
		n.temp, err = prepareReplacement(target, n.content)
		return err
	case saveprotocol.Lock:
		if n.lock == nil {
			path, err := lockPath(n.target)
			if err != nil {
				return err
			}
			n.lock = flock.New(path)
		}
		locked, err := n.lock.TryLock()
		if err != nil {
			return fmt.Errorf("lock file: %w", err)
		}
		if !locked {
			return fmt.Errorf("%w: %s", ErrFileBusy, n.target)
		}
	case saveprotocol.Validate:
		current, content, _, err := readCurrentRevision(n.path)
		if err != nil {
			return err
		}
		// Re-resolve the logical path, not just the prepared target. This also
		// rejects a symlink retarget during a force-save's preparation.
		if current.target != n.target || (n.expected != nil && !n.expected.equal(current)) {
			return &ConflictError{Path: n.path, DiskContent: content}
		}
		n.current, n.diskContent = current, content
	case saveprotocol.CaptureBefore:
		if n.current.exists && n.store.OnRead != nil {
			if err := n.store.OnRead(n.target, n.diskContent); err != nil {
				return fmt.Errorf("capture overwritten version: %w", err)
			}
		}
	case saveprotocol.Replace:
		if err := renameFile(n.temp, n.target); err != nil {
			return fmt.Errorf("replace file: %w", err)
		}
		n.temp = ""
		// Replacement is the commit point, even if later sync/history/unlock fails.
		if n.model != nil {
			var modTime time.Time
			if info, err := os.Stat(n.target); err == nil {
				modTime = info.ModTime()
			}
			n.model.setRevision(fileRevision{target: n.target, exists: true, hash: sha256.Sum256([]byte(n.content))}, modTime)
		}
	case saveprotocol.SyncDirectory:
		return syncParentDirectory(filepath.Dir(n.target))
	case saveprotocol.CaptureAfter:
		if n.store.OnWrite != nil {
			return n.store.OnWrite(n.target, n.content)
		}
	case saveprotocol.Unlock:
		return n.lock.Unlock()
	case saveprotocol.Done:
		return errors.New("completed save scheduled")
	}
	return nil
}
func (n *nativeSave) cleanup() error {
	var err error
	if n.temp != "" {
		if removeErr := os.Remove(n.temp); !errors.Is(removeErr, fs.ErrNotExist) {
			err = removeErr
		}
	}
	if n.lock != nil {
		err = errors.Join(err, n.lock.Close())
	}
	return err
}

func (store Store) writeContent(filePath, content string, expected *fileRevision, fm *FileModel, force bool) (err error) {
	driver := &nativeSave{store: store, path: filePath, content: content, expected: expected, model: fm}
	save := saveprotocol.New(force, lockWait)
	started := time.Now()
	defer func() {
		err = errors.Join(err, driver.cleanup())
		if err != nil && save.Outcome().Committed {
			err = &PostCommitError{Err: err}
		}
	}()
	for save.Phase() != saveprotocol.Done {
		phase := save.Phase()
		effectErr := driver.execute(phase)
		save.Advance(effectErr, time.Since(started))
		if effectErr == nil && saveStageHook != nil {
			saveStageHook(stageName(phase))
		}
		if errors.Is(effectErr, ErrFileBusy) && save.Phase() == saveprotocol.Lock {
			time.Sleep(lockRetry)
		}
	}
	return save.Outcome().Err
}
func stageName(phase saveprotocol.Phase) string {
	switch phase {
	case saveprotocol.Prepare:
		return "prepared"
	case saveprotocol.Lock:
		return "locked"
	case saveprotocol.Validate:
		return "validated"
	case saveprotocol.CaptureBefore:
		return "captured-before"
	case saveprotocol.Replace:
		return "replaced"
	case saveprotocol.SyncDirectory:
		return "synced"
	case saveprotocol.CaptureAfter:
		return "captured-after"
	case saveprotocol.Unlock:
		return "unlocked"
	default:
		return "done"
	}
}

// HasUnsavedContent compares the editable document with its loaded disk revision.
// It is used to avoid losing read-only checklist edits during watcher reloads.
func (fm *FileModel) HasUnsavedContent() bool {
	if !fm.revisionKnown {
		return false
	}
	content := SerializeMarkdown(fm)
	if !fm.revision.exists {
		return content != "# Todos\n\n"
	}
	return sha256.Sum256([]byte(content)) != fm.revision.hash
}
