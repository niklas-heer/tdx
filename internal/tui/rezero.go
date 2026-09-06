package tui

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/niklas-heer/tdx/internal/editor"
	"github.com/niklas-heer/tdx/internal/markdown"
)

type rezeroRound struct {
	Marked   int
	Phase    string // review, work, complete; empty outside Rezero
	Review   []int  // fixed bottom-to-top snapshot for this round
	Position int
	Ready    map[int]bool
	Work     []int
}

func (r rezeroRound) clone() rezeroRound {
	r.Review = slices.Clone(r.Review)
	r.Work = slices.Clone(r.Work)
	r.Ready = maps.Clone(r.Ready)
	return r
}

type rezeroUndo struct {
	Document *markdown.FileModel
	Round    rezeroRound
	Selected int
}

func (m *Model) startRezero() {
	if m.ConflictPending {
		m.Err = fmt.Errorf("resolve the file conflict before starting Rezero")
		return
	}
	m.rezero = rezeroRound{Phase: "review", Ready: make(map[int]bool)}
	m.rezeroUndo = nil
	m.NumberBuffer = ""
	m.gPressed = false
	m.nextRezeroRound()
	m.InvalidateDocumentTree()
}

func (m *Model) stopRezero() {
	m.rezero = rezeroRound{}
	m.rezeroHelp = false
	m.rezeroUndo = nil
	m.rezeroInput = ""
	m.InputMode, m.EditMode = false, false
	m.InvalidateHeadingsCache()
	m.AvailableTags = markdown.GetAllTags(m.FileModel.Todos)
	m.AvailablePriorities = markdown.GetAllPriorities(m.FileModel.Todos)
	m.InvalidateDocumentTree()
	m.adjustSelectionForFilter()
}

func (m *Model) nextRezeroRound() {
	m.rezero = rezeroRound{Phase: "review", Ready: make(map[int]bool)}
	for i := len(m.FileModel.Todos) - 1; i >= 0; i-- {
		if !m.FileModel.Todos[i].Checked {
			m.rezero.Review = append(m.rezero.Review, i)
		}
	}
	if len(m.rezero.Review) == 0 {
		m.rezero.Phase = "complete"
		return
	}
	m.SelectedIndex = m.rezero.Review[0]
}

func (m *Model) reviewRezero(ready bool) {
	r := &m.rezero
	if r.Position >= len(r.Review) {
		return
	}
	index := r.Review[r.Position]
	if ready {
		r.Ready[index] = true
	} else {
		delete(r.Ready, index)
	}
	r.Position++
	if r.Position < len(r.Review) {
		m.SelectedIndex = r.Review[r.Position]
		return
	}
	r.Phase = "work"
	r.Marked = len(r.Ready)
	for _, i := range r.Review {
		if r.Ready[i] {
			r.Work = append(r.Work, i)
		}
	}
	m.advanceRezero()
}

func (m *Model) advanceRezero() {
	r := &m.rezero
	for len(r.Work) > 0 {
		i := r.Work[0]
		if i < len(m.FileModel.Todos) && !m.FileModel.Todos[i].Checked {
			m.SelectedIndex = i
			return
		}
		delete(r.Ready, i)
		r.Work = r.Work[1:]
	}
	r.Phase = "complete"
}

func (m Model) handleRezeroKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	m.rezero = m.rezero.clone()
	key := msg.String()
	switch key {
	case "esc":
		m.stopRezero()
		return m, nil
	case "ctrl+c", "q":
		return m, tea.Quit
	case ":":
		m.CommandMode = true
		m.InputBuffer = ""
		m.CursorPos = 0
		m.CommandCursor = 0
		m.updateFilteredCommands()
		return m, nil
	case "?":
		m.rezeroHelp = !m.rezeroHelp
		return m, nil
	}
	if m.ConflictPending {
		m.Err = fmt.Errorf("resolve the conflict with :reload or :force-save before continuing")
		return m, nil
	}
	switch key {
	case "u":
		m.undoRezero()
		return m, nil
	case "n", "N":
		if m.ReadOnly {
			m.Err = fmt.Errorf("read-only file: task creation is disabled")
			return m, nil
		}
		m.rezeroInput = "new"
		m.InputMode = true
		m.InsertAfterCursor = false
		m.InputBuffer = ""
		m.CursorPos = 0
		return m, nil
	}
	switch m.rezero.Phase {
	case "review":
		switch key {
		case "space", ".":
			m.reviewRezero(true)
		case "enter", "up", "k":
			m.reviewRezero(false)
		case "b", "down", "j":
			if m.rezero.Position > 0 {
				m.rezero.Position--
				m.SelectedIndex = m.rezero.Review[m.rezero.Position]
				delete(m.rezero.Ready, m.SelectedIndex)
			}
		}
	case "work":
		switch key {
		case "space", "enter":
			if m.commitRezero(editor.Action{Kind: editor.SetChecked, Index: m.SelectedIndex, Checked: true}) {
				m.advanceRezero()
			}
		case "r":
			if m.ReadOnly {
				m.Err = fmt.Errorf("read-only file: continuation is disabled")
				return m, nil
			}
			title, err := m.FileModel.ContinuationTitle(m.SelectedIndex)
			if err != nil {
				m.Err = err
				return m, nil
			}
			m.rezeroInput = "continue"
			m.EditMode = true
			m.InputBuffer = title
			m.CursorPos = len(title)
		case "p": // Defer a selected task without marking it complete.
			delete(m.rezero.Ready, m.SelectedIndex)
			m.rezero.Work = m.rezero.Work[1:]
			m.advanceRezero()
		}
	case "complete":
		if key == "enter" || key == "r" {
			m.nextRezeroRound()
		}
	}
	return m, nil
}

func (m Model) handleRezeroInput(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if msg.Text != "" {
		return m.handleInputKey(msg)
	}
	switch msg.String() {
	case "esc":
		m.InputMode, m.EditMode = false, false
		m.rezeroInput = ""
		m.InputBuffer = ""
		m.CursorPos = 0
	case "enter", "ctrl+m":
		kind := editor.AppendSource
		if m.rezeroInput == "continue" {
			kind = editor.Continue
		}
		if m.commitRezero(editor.Action{Kind: kind, Index: m.SelectedIndex, Text: m.InputBuffer}) {
			m.rezero = m.rezero.clone()
			if kind == editor.Continue {
				m.advanceRezero()
			}
			m.InputMode, m.EditMode = false, false
			m.rezeroInput = ""
			m.InputBuffer = ""
			m.CursorPos = 0
		}
	default:
		return m.handleInputKey(msg)
	}
	return m, nil
}

// Save the complete candidate before advancing the round. A precommit failure
// leaves both the current document and input available for retry. Conflicts also
// retain the candidate in the existing explicit diff/force-save flow.
func (m *Model) commitRezero(action editor.Action) bool {
	if m.ReadOnly {
		m.Err = fmt.Errorf("read-only file: Rezero edits are disabled")
		return false
	}
	if m.ConflictPending {
		m.Err = fmt.Errorf("resolve the file conflict before editing")
		return false
	}
	candidate := m.FileModel.Clone()
	if _, err := editor.Apply(candidate, action); err != nil {
		m.Err = err
		return false
	}
	if !m.saveRezeroCandidate(candidate) {
		return false
	}
	entry := rezeroUndo{Document: m.FileModel.Clone(), Round: m.rezero.clone(), Selected: m.SelectedIndex}
	m.history.Push(&m.FileModel)
	if len(m.rezeroUndo) == editor.HistoryLimit {
		m.rezeroUndo = slices.Clone(m.rezeroUndo[1:])
	}
	m.rezeroUndo = append(m.rezeroUndo, entry)
	m.FileModel = *candidate
	m.AvailableTags = markdown.GetAllTags(m.FileModel.Todos)
	m.AvailablePriorities = markdown.GetAllPriorities(m.FileModel.Todos)
	m.InvalidateHeadingsCache()
	m.InvalidateDocumentTree()
	// Do not prune temporarily bypassed filters; restore them unchanged on exit.
	return true
}

func (m *Model) saveRezeroCandidate(candidate *markdown.FileModel) bool {
	err := m.Config().Store.WriteFile(m.FilePath, candidate)
	var postCommit *markdown.PostCommitError
	if err != nil && !errors.As(err, &postCommit) {
		m.recordSaveError(err, markdown.SerializeMarkdown(candidate))
		return false
	}
	m.clearConflict()
	m.Err = err
	return true
}

func (m *Model) undoRezero() {
	if m.ReadOnly {
		m.Err = fmt.Errorf("read-only file: undo is disabled")
		return
	}
	if len(m.rezeroUndo) == 0 {
		return
	}
	entry := m.rezeroUndo[len(m.rezeroUndo)-1]
	candidate := m.FileModel.Clone()
	candidate.RestoreContent(entry.Document)
	if !m.saveRezeroCandidate(candidate) {
		return
	}
	m.FileModel = *candidate
	m.AvailableTags = markdown.GetAllTags(m.FileModel.Todos)
	m.AvailablePriorities = markdown.GetAllPriorities(m.FileModel.Todos)
	// Keep the ordinary undo stack aligned for users who later leave Rezero.
	m.history.Undo(&m.FileModel)
	m.rezeroUndo = m.rezeroUndo[:len(m.rezeroUndo)-1]
	m.rezero = entry.Round.clone()
	m.SelectedIndex = entry.Selected
	m.InvalidateHeadingsCache()
	m.InvalidateDocumentTree()
}

func (m Model) rezeroStatus() string {
	var status, hints, primary string
	switch {
	case m.rezeroInput == "continue":
		status = "REZERO CONTINUE"
		hints = "enter re-enter at file end · esc cancel"
		primary = "enter save · esc cancel"
	case m.rezeroInput == "new":
		status = "REZERO NEW"
		hints = "enter add for next round · esc cancel"
		primary = "enter save · esc cancel"
	case m.rezero.Phase == "review":
		status = fmt.Sprintf("REZERO REVIEW %d/%d · %d marked", m.rezero.Position, len(m.rezero.Review), len(m.rezero.Ready))
		hints = "space mark · enter pass · b back · n new · esc exit"
		primary = "space mark · enter pass · ?"
	case m.rezero.Phase == "work":
		status = fmt.Sprintf("REZERO WORK · %d left", len(m.rezero.Work))
		hints = "space done · r continue · p defer · n new · u undo · esc exit"
		primary = "space done · r continue · ?"
	default:
		status = "REZERO ROUND COMPLETE"
		if len(m.rezero.Review) == 0 {
			status = "REZERO · no open tasks"
		} else if m.rezero.Marked == 0 {
			status = "REZERO · none selected"
		}
		hints = "enter review again · n new · u undo · esc exit"
		primary = "enter review · esc exit · ?"
	}
	if m.ReadOnly {
		status = "READ ONLY · " + status
	}
	text := status + "  " + hints + " · ? keys"
	if m.TermWidth > 0 && ansi.StringWidth(text) > m.TermWidth {
		text = status + "  " + primary
		if ansi.StringWidth(text) > m.TermWidth {
			short := strings.TrimPrefix(status, "REZERO ")
			if m.rezero.Phase == "review" && m.rezeroInput == "" {
				short = fmt.Sprintf("REVIEW %d/%d", m.rezero.Position, len(m.rezero.Review))
			}
			if m.ReadOnly {
				short = "RO " + strings.TrimPrefix(short, "READ ONLY · REZERO ")
			}
			text = ansi.Truncate(short, max(0, m.TermWidth-4), "") + " · ?"
			text = ansi.Truncate(text, m.TermWidth, "")
		}
	}
	if m.rezeroInput != "" {
		text = status + "  " + hints
		if m.TermWidth > 0 && ansi.StringWidth(text) > m.TermWidth {
			text = ansi.Truncate(primary, m.TermWidth, "…")
		}
	}
	if m.rezeroHelp {
		width := m.TermWidth
		if width <= 0 {
			width = 80
		}
		text += "\n" + ansi.Hardwrap(hints+"\nFull file order; filters paused. New tasks join next round. Dots last for this session. Continue retires the subtree and appends its copy at file end.", width, true)
	}
	return strings.TrimRight(text, "\n")
}
