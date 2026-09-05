package markdown

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
	"github.com/niklas-heer/tdx/internal/config"
)

var (
	// ErrFileChanged identifies a conditional save rejected because disk changed.
	ErrFileChanged = errors.New("file changed externally")
	// ErrFileBusy identifies a save that could not acquire the tdx lock in time.
	ErrFileBusy = errors.New("file is busy in another tdx process")
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
	content, err := os.ReadFile(target)
	if errors.Is(err, fs.ErrNotExist) {
		return fileRevision{target: target}, "", time.Time{}, nil
	}
	if err != nil {
		return fileRevision{}, "", time.Time{}, fmt.Errorf("read %q: %w", target, err)
	}
	info, err := os.Stat(target)
	if err != nil {
		return fileRevision{}, "", time.Time{}, fmt.Errorf("stat %q: %w", target, err)
	}
	if !info.Mode().IsRegular() {
		return fileRevision{}, "", time.Time{}, fmt.Errorf("%q is not a regular file", target)
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
			if closeErr := tmp.Close(); closeErr != nil && err == nil {
				err = fmt.Errorf("close replacement: %w", closeErr)
			}
		}
		if err != nil {
			_ = os.Remove(name)
		}
	}()

	if preserveMode {
		if err = tmp.Chmod(mode); err != nil {
			return "", fmt.Errorf("set replacement permissions: %w", err)
		}
	}
	if _, err = tmp.WriteString(content); err != nil {
		return "", fmt.Errorf("write replacement: %w", err)
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

func (store Store) writeContent(filePath, content string, expected *fileRevision, fm *FileModel, force bool) (err error) {
	target, err := resolveTarget(filePath)
	if err != nil {
		return err
	}
	tmpPath, err := prepareReplacement(target, content)
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmpPath) }()
	if saveStageHook != nil {
		saveStageHook("prepared")
	}

	path, err := lockPath(target)
	if err != nil {
		return err
	}
	fileLock := flock.New(path)
	ctx, cancel := context.WithTimeout(context.Background(), lockWait)
	defer cancel()
	locked, err := fileLock.TryLockContext(ctx, lockRetry)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("%w: %s", ErrFileBusy, target)
		}
		return fmt.Errorf("lock file: %w", err)
	}
	if !locked {
		return fmt.Errorf("%w: %s", ErrFileBusy, target)
	}
	committed := false
	defer func() {
		if unlockErr := fileLock.Unlock(); unlockErr != nil {
			if err == nil {
				if committed {
					err = &PostCommitError{Err: unlockErr}
				} else {
					err = unlockErr
				}
			} else {
				err = errors.Join(err, unlockErr)
			}
		}
	}()

	current, diskContent, _, err := readCurrentRevision(target)
	if err != nil {
		return err
	}
	if expected != nil && !expected.equal(current) {
		return &ConflictError{Path: target, DiskContent: diskContent}
	}
	if force && current.exists && store.OnRead != nil {
		if err := store.OnRead(target, diskContent); err != nil {
			return fmt.Errorf("capture overwritten version: %w", err)
		}
	}

	if err := renameFile(tmpPath, target); err != nil {
		return fmt.Errorf("replace file: %w", err)
	}
	committed = true
	tmpPath = ""
	if saveStageHook != nil {
		saveStageHook("replaced")
	}

	newRevision := fileRevision{target: target, exists: true, hash: sha256.Sum256([]byte(content))}
	var modTime time.Time
	if info, statErr := os.Stat(target); statErr == nil {
		modTime = info.ModTime()
	}
	if fm != nil {
		fm.setRevision(newRevision, modTime)
	}

	var postCommitErr error
	if syncErr := syncParentDirectory(filepath.Dir(target)); syncErr != nil {
		postCommitErr = fmt.Errorf("sync parent directory: %w", syncErr)
	}
	if store.OnWrite != nil {
		if hookErr := store.OnWrite(target, content); hookErr != nil {
			postCommitErr = errors.Join(postCommitErr, hookErr)
		}
	}
	if postCommitErr != nil {
		return &PostCommitError{Err: postCommitErr}
	}
	return nil
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
