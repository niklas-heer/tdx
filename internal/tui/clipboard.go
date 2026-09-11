package tui

import "github.com/niklas-heer/tdx/internal/util"

type clipboardCopiedMsg struct{ err error }
type clipboardPastedMsg struct {
	text               string
	err                error
	file, mode, buffer string
	cursor             int
}

func (m *Model) clipboard() util.Clipboard {
	if c := m.Config().Clipboard; c != nil {
		return c
	}
	return util.SystemClipboard{}
}
func (m *Model) clipboardInputMode() string {
	switch {
	case m.ViewMode != "":
		return "view:" + m.ViewMode
	case m.HeadingInput != "":
		return "heading:" + m.HeadingInput
	case m.InputMode:
		return "new"
	case m.EditMode:
		return "edit"
	case m.SearchMode:
		return "search"
	case m.CommandMode:
		return "command"
	}
	return ""
}
