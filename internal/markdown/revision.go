package markdown

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// RevisionToken identifies the exact bytes loaded from disk, including frontmatter.
// A missing file is distinct from an existing empty file. In-memory edits do not
// change the token until a successful save establishes a new disk revision.
func (fm *FileModel) RevisionToken() (string, error) {
	if !fm.revisionKnown {
		return "", ErrRevisionUnknown
	}
	if !fm.revision.exists {
		return "missing", nil
	}
	return "sha256:" + hex.EncodeToString(fm.revision.hash[:]), nil
}

func ValidateRevisionToken(token string) error {
	if token == "missing" {
		return nil
	}
	hash, ok := strings.CutPrefix(token, "sha256:")
	decoded, err := hex.DecodeString(hash)
	if !ok || err != nil || len(decoded) != sha256.Size || hash != strings.ToLower(hash) {
		return fmt.Errorf("revision must be 'missing' or sha256:<64 lowercase hex digits>")
	}
	return nil
}

// RequireRevision checks a caller's snapshot before applying an edit. WriteFile
// still checks the loaded revision under its lock to reject subsequent changes.
func (fm *FileModel) RequireRevision(expected string) error {
	if err := ValidateRevisionToken(expected); err != nil {
		return err
	}
	actual, err := fm.RevisionToken()
	if err != nil {
		return err
	}
	if actual != expected {
		return fmt.Errorf("%w: revision does not match; query again before editing", ErrFileChanged)
	}
	return nil
}
