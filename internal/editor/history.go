package editor

import "github.com/niklas-heer/tdx/internal/markdown"

const HistoryLimit = 100

// History holds snapshots for a single editing session. Its zero value is ready to use.
type History struct{ past []*markdown.FileModel }

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
func (h *History) Clear()   { h.past = nil }
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
