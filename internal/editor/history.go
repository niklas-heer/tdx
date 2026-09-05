package editor

import "github.com/niklas-heer/tdx/internal/markdown"

const HistoryLimit = 100

// History holds snapshots for a single editing session. Its zero value is ready to use.
type History struct {
	past    []*markdown.FileModel
	pending *markdown.FileModel
}

func (h *History) Push(doc *markdown.FileModel) {
	if doc == nil {
		return
	}
	if len(h.past) == HistoryLimit {
		copy(h.past, h.past[1:])
		h.past[len(h.past)-1] = nil
		h.past = h.past[:len(h.past)-1]
	}
	h.past = append(h.past, doc.Clone())
}
func (h *History) Len() int { return len(h.past) }
func (h *History) Clear()   { h.past = nil; h.pending = nil }
func (h *History) Undo(doc *markdown.FileModel) bool {
	if doc == nil || len(h.past) == 0 {
		return false
	}
	last := len(h.past) - 1
	doc.RestoreContent(h.past[last])
	h.past[last] = nil
	h.past = h.past[:last]
	return true
}

// Begin keeps a provisional snapshot outside the bounded undo stack. Cancelling
// input must not evict the oldest committed edit when history is full.
func (h *History) Begin(doc *markdown.FileModel) { h.pending = doc.Clone() }
func (h *History) Commit() {
	if h.pending == nil {
		return
	}
	if len(h.past) == HistoryLimit {
		copy(h.past, h.past[1:])
		h.past = h.past[:HistoryLimit-1]
	}
	h.past = append(h.past, h.pending)
	h.pending = nil
}
func (h *History) Cancel(doc *markdown.FileModel) bool {
	if h.pending == nil || doc == nil {
		return false
	}
	doc.RestoreContent(h.pending)
	h.pending = nil
	return true
}

// CommitIfChanged discards no-op gestures without consuming bounded undo space.
func (h *History) CommitIfChanged(doc *markdown.FileModel) {
	if h.pending == nil {
		return
	}
	if markdown.SerializeMarkdown(h.pending) == markdown.SerializeMarkdown(doc) {
		h.pending = nil
		return
	}
	h.Commit()
}
