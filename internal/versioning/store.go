// Package versioning provides SQLite-backed continuous versioning for markdown files.
// All files share a single database (versions.sqlite) stored in the tdx config directory.
package versioning

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/niklas-heer/tdx/internal/config"
	_ "modernc.org/sqlite"
)

// VersionInfo holds metadata for a single stored version of a file.
type VersionInfo struct {
	ID        int64
	CreatedAt time.Time
}

// zstdEncoder and zstdDecoder are package-level singletons initialised once.
// Using EncodeAll/DecodeAll for in-memory compression avoids per-call setup overhead.
var (
	zstdEncoder *zstd.Encoder
	zstdDecoder *zstd.Decoder
)

func init() {
	var err error
	zstdEncoder, err = zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedFastest))
	if err != nil {
		panic("versioning: init zstd encoder: " + err.Error())
	}
	zstdDecoder, err = zstd.NewReader(nil, zstd.WithDecoderConcurrency(1))
	if err != nil {
		panic("versioning: init zstd decoder: " + err.Error())
	}
}

// getStoreDir is a variable so it can be overridden in tests.
var getStoreDir = config.GetConfigDir

// originalGetStoreDir stores the original function for reset.
var originalGetStoreDir = config.GetConfigDir

// SetStoreDirForTesting overrides the directory used for the SQLite database in tests.
func SetStoreDirForTesting(dir string) {
	getStoreDir = func() (string, error) {
		return dir, nil
	}
}

// ResetStoreDirForTesting restores the default store directory function.
func ResetStoreDirForTesting() {
	getStoreDir = originalGetStoreDir
}

// DBPath returns the path to the shared versions.sqlite database.
func DBPath() (string, error) {
	storeDir, err := getStoreDir()
	if err != nil {
		return "", fmt.Errorf("versioning: get store dir: %w", err)
	}
	return filepath.Join(storeDir, "versions.sqlite"), nil
}

// Store holds an open shared SQLite database for all markdown files.
type Store struct {
	db          *sql.DB
	dbPath      string
	fileIDCache map[string]int64 // filePath → files.id cache for this session
	MaxVersions int              // max versions per file (≤0 = unlimited)
}

const initSQL = `
CREATE TABLE IF NOT EXISTS files (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    file_path TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS file_versions (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    file_id        INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    version_hash   TEXT NOT NULL,
    content        BLOB NOT NULL,
    created_at     DATETIME DEFAULT CURRENT_TIMESTAMP,
    commit_message TEXT,
    UNIQUE(file_id, version_hash)
);

CREATE INDEX IF NOT EXISTS idx_file_versions_file_id ON file_versions(file_id);
`

// Open opens (or creates) the shared version store.
// maxVersions controls the per-file retention limit (≤0 = unlimited).
func Open(maxVersions int) (*Store, error) {
	dbPath, err := DBPath()
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("versioning: create store dir: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("versioning: open db: %w", err)
	}

	// Optimise for high-frequency writes.
	if _, err := db.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("versioning: set WAL mode: %w", err)
	}
	if _, err := db.Exec("PRAGMA synchronous=NORMAL;"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("versioning: set synchronous: %w", err)
	}

	if _, err := db.Exec(initSQL); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("versioning: init schema: %w", err)
	}

	return &Store{
		db:          db,
		dbPath:      dbPath,
		fileIDCache: make(map[string]int64),
		MaxVersions: maxVersions,
	}, nil
}

// resolveFileID returns the files.id for filePath, inserting a row if necessary.
// Results are cached for the lifetime of the Store to avoid repeated DB round-trips.
func (s *Store) resolveFileID(filePath string) (int64, error) {
	if id, ok := s.fileIDCache[filePath]; ok {
		return id, nil
	}
	if _, err := s.db.Exec(`INSERT OR IGNORE INTO files (file_path) VALUES (?)`, filePath); err != nil {
		return 0, fmt.Errorf("versioning: upsert file path: %w", err)
	}
	var id int64
	if err := s.db.QueryRow(`SELECT id FROM files WHERE file_path = ?`, filePath).Scan(&id); err != nil {
		return 0, fmt.Errorf("versioning: resolve file id: %w", err)
	}
	s.fileIDCache[filePath] = id
	return id, nil
}

// SaveVersion inserts a zstd-compressed snapshot of content for the given file.
// The version_hash is computed from the uncompressed content so deduplication is hash-based.
// If an identical version (same SHA-256 hash) already exists for that file the call is a no-op.
func (s *Store) SaveVersion(filePath, content string) error {
	fileID, err := s.resolveFileID(filePath)
	if err != nil {
		return err
	}
	hash := sha256sum(content)
	compressed := zstdEncoder.EncodeAll([]byte(content), nil)
	_, err = s.db.Exec(
		`INSERT OR IGNORE INTO file_versions (file_id, version_hash, content) VALUES (?, ?, ?)`,
		fileID, hash, compressed,
	)
	if err != nil {
		return fmt.Errorf("versioning: save version: %w", err)
	}
	return nil
}

// ReadVersion returns the decompressed content stored in a specific version row for filePath.
// id is the file_versions.id value.
func (s *Store) ReadVersion(filePath string, id int64) (string, error) {
	fileID, err := s.resolveFileID(filePath)
	if err != nil {
		return "", err
	}
	var compressed []byte
	err = s.db.QueryRow(
		`SELECT content FROM file_versions WHERE file_id = ? AND id = ?`,
		fileID, id,
	).Scan(&compressed)
	if err != nil {
		return "", fmt.Errorf("versioning: read version: %w", err)
	}
	decompressed, err := zstdDecoder.DecodeAll(compressed, nil)
	if err != nil {
		return "", fmt.Errorf("versioning: decompress version: %w", err)
	}
	return string(decompressed), nil
}

// Prune removes the oldest versions for filePath, keeping at most maxVersions rows.
// If maxVersions ≤ 0 the call is a no-op.
func (s *Store) Prune(filePath string, maxVersions int) error {
	if maxVersions <= 0 {
		return nil
	}
	fileID, err := s.resolveFileID(filePath)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		DELETE FROM file_versions
		WHERE file_id = ?
		  AND id NOT IN (
		    SELECT id FROM file_versions WHERE file_id = ? ORDER BY id DESC LIMIT ?
		  )`, fileID, fileID, maxVersions)
	if err != nil {
		return fmt.Errorf("versioning: prune: %w", err)
	}
	return nil
}

// PruneAll prunes every file path seen this session using maxVersions.
// Intended to be called before Close.
func (s *Store) PruneAll(maxVersions int) {
	for filePath := range s.fileIDCache {
		_ = s.Prune(filePath, maxVersions)
	}
}

// ListVersions returns the version history for filePath ordered most-recent-first.
// Returns an empty slice (not an error) if no versions exist for the file.
func (s *Store) ListVersions(filePath string) (versions []VersionInfo, err error) {
	fileID, err := s.resolveFileID(filePath)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Query(
		`SELECT id, created_at FROM file_versions WHERE file_id = ? ORDER BY id DESC`,
		fileID,
	)
	if err != nil {
		return nil, fmt.Errorf("versioning: list versions: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("versioning: close version rows: %w", closeErr)
		}
	}()
	for rows.Next() {
		var v VersionInfo
		var createdAt string
		if err := rows.Scan(&v.ID, &createdAt); err != nil {
			return nil, fmt.Errorf("versioning: list versions scan: %w", err)
		}
		// modernc.org/sqlite returns CURRENT_TIMESTAMP as RFC3339 ("2006-01-02T15:04:05Z").
		// Try that first, then the classic SQLite space-separated format as a fallback.
		for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05"} {
			if t, err := time.Parse(layout, createdAt); err == nil {
				v.CreatedAt = t
				break
			}
		}
		versions = append(versions, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("versioning: list versions rows: %w", err)
	}
	if versions == nil {
		versions = []VersionInfo{}
	}
	return versions, err
}

// Close checkpoints the WAL back into the main database file, then closes the
// connection. Without an explicit checkpoint, SQLite leaves the .wal and .shm
// files on disk even after the last connection is closed.
func (s *Store) Close() error {
	// TRUNCATE mode writes all WAL frames to the database and truncates the WAL
	// file to zero bytes, leaving a clean single-file state on exit.
	_, _ = s.db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
	return s.db.Close()
}

func sha256sum(str string) string {
	h := sha256.Sum256([]byte(str))
	return hex.EncodeToString(h[:])
}
