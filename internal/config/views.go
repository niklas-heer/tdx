package config

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// SectionRef identifies a section by its heading ancestry, surviving insertions.
// Occurrence disambiguates repeated headings with the same ancestry.
type SectionRef struct {
	Path       []string `json:"path"`
	Occurrence int      `json:"occurrence"`
}
type SavedView struct {
	Tags         []string     `json:"tags,omitempty"`
	Priorities   []int        `json:"priorities,omitempty"`
	Due          string       `json:"due,omitempty"`
	FilterDone   bool         `json:"filter_done"`
	Focus        *SectionRef  `json:"focus,omitempty"`
	Folded       []SectionRef `json:"folded,omitempty"`
	ShowHeadings bool         `json:"show_headings"`
}
type SavedViews struct {
	Active string               `json:"active,omitempty"`
	Views  map[string]SavedView `json:"views"`
}
type ViewStore interface {
	Load(string) (*SavedViews, error)
	Save(string, *SavedViews) error
}
type FileViewStore struct{ Dir string }

func NewFileViewStore(dir string) ViewStore { return FileViewStore{Dir: dir} }
func (s FileViewStore) path(file string) (string, error) {
	canonical, err := filepath.Abs(file)
	if err != nil {
		return "", err
	}
	// Resolve the existing ancestor too, so new files beneath a symlinked
	// directory share the same key before and after creation.
	unresolved := []string{}
	for {
		resolved, resolveErr := filepath.EvalSymlinks(canonical)
		if resolveErr == nil {
			canonical = resolved
			break
		}
		if !os.IsNotExist(resolveErr) {
			return "", resolveErr
		}
		parent := filepath.Dir(canonical)
		if parent == canonical {
			return "", resolveErr
		}
		unresolved = append(unresolved, filepath.Base(canonical))
		canonical = parent
	}
	for i := len(unresolved) - 1; i >= 0; i-- {
		canonical = filepath.Join(canonical, unresolved[i])
	}
	sum := sha256.Sum256([]byte(filepath.Clean(canonical)))
	dir := s.Dir
	if dir == "" {
		dir, err = GetConfigDir()
		if err != nil {
			return "", err
		}
	}
	return filepath.Join(dir, "views", hex.EncodeToString(sum[:])+".json"), nil
}
func (s FileViewStore) Load(file string) (*SavedViews, error) {
	path, err := s.path(file)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &SavedViews{Views: map[string]SavedView{}}, nil
	}
	if err != nil {
		return nil, err
	}
	var state SavedViews
	if err = json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("read saved views: %w", err)
	}
	if state.Views == nil {
		state.Views = map[string]SavedView{}
	}
	if err := validateViews(&state); err != nil {
		return nil, err
	}
	return &state, nil
}
func (s FileViewStore) Save(file string, state *SavedViews) error {
	path, err := s.path(file)
	if err != nil {
		return err
	}
	if err := validateViews(state); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".views-*")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	if _, err = temp.Write(data); err != nil {
		temp.Close()
		return err
	}
	if err = temp.Sync(); err != nil {
		temp.Close()
		return err
	}
	if err = temp.Close(); err != nil {
		return err
	}
	return os.Rename(temp.Name(), path)
}

func validateViews(state *SavedViews) error {
	if state == nil {
		return fmt.Errorf("saved views cannot be nil")
	}
	validRef := func(ref SectionRef) bool {
		if len(ref.Path) == 0 || ref.Occurrence < 1 {
			return false
		}
		for _, part := range ref.Path {
			if strings.TrimSpace(part) == "" {
				return false
			}
		}
		return true
	}
	for name, view := range state.Views {
		if strings.TrimSpace(name) == "" || strings.IndexFunc(name, unicode.IsControl) >= 0 {
			return fmt.Errorf("view name must contain text without control characters")
		}
		switch view.Due {
		case "", "all", "today", "week", "overdue":
		default:
			return fmt.Errorf("saved view %q has invalid due filter", name)
		}
		for _, p := range view.Priorities {
			if p < 1 {
				return fmt.Errorf("saved view %q has invalid priority", name)
			}
		}
		if view.Focus != nil && !validRef(*view.Focus) {
			return fmt.Errorf("saved view %q has invalid section focus", name)
		}
		for _, fold := range view.Folded {
			if !validRef(fold) {
				return fmt.Errorf("saved view %q has invalid section fold", name)
			}
		}
	}
	return nil
}
