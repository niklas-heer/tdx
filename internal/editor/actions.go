// Package editor defines document edits independently of terminal input and persistence.
package editor

import (
	"fmt"

	"github.com/niklas-heer/tdx/internal/markdown"
)

type Kind string

const (
	Add            Kind = "add"
	Insert         Kind = "insert"
	AddInSection   Kind = "add-in-section"
	Edit           Kind = "edit"
	Toggle         Kind = "toggle"
	SetChecked     Kind = "set-checked"
	Delete         Kind = "delete"
	Move           Kind = "move"
	MoveToPosition Kind = "move-to-position"
	Indent         Kind = "indent"
	Outdent        Kind = "outdent"
	RenameHeading  Kind = "rename-heading"
	CreateHeading  Kind = "create-heading"
	SetAllChecked  Kind = "set-all-checked"
	ClearDone      Kind = "clear-done"
	SortDone       Kind = "sort-done"
	SortDue        Kind = "sort-due"
	SortPriority   Kind = "sort-priority"
)

// Action uses zero-based document indexes. UI filtering never changes these indexes.
// Index selects a task, or a heading for heading actions; Target is a move destination.
// CreateHeading uses Index=-1 to append a root heading.
type Action struct {
	Kind        Kind
	Index       int
	Target      int
	Text        string
	Checked     bool
	InsertAfter bool
	Level       int
}

// Apply validates and applies one document action, returning the affected index.
// It does not save, change selection, or capture undo; callers group interactive edits.
func Apply(doc *markdown.FileModel, action Action) (int, error) {
	if doc == nil {
		return -1, fmt.Errorf("document is unavailable")
	}
	index := action.Index
	switch action.Kind {
	case Edit, Toggle, SetChecked, Delete, Move, MoveToPosition, Indent, Outdent, Insert:
		if index < 0 || index >= len(doc.Todos) {
			return -1, fmt.Errorf("invalid todo index: %d", index)
		}
	}
	switch action.Kind {
	case Add:
		err := doc.AddTodoItemChecked(action.Text, action.Checked)
		return len(doc.Todos) - 1, err
	case Insert:
		return doc.InsertTodoItemAfterChecked(index, action.Text, action.Checked)
	case AddInSection:
		return doc.AddTodoInSection(index, action.Text)
	case Edit:
		return index, doc.UpdateTodoItem(index, action.Text, doc.Todos[index].Checked)
	case Toggle:
		return index, doc.UpdateTodoItem(index, doc.Todos[index].Text, !doc.Todos[index].Checked)
	case SetChecked:
		return index, doc.UpdateTodoItem(index, doc.Todos[index].Text, action.Checked)
	case Delete:
		return index, doc.DeleteTodoItem(index)
	case Move:
		moved := moveResultIndex(doc.Todos, index, action.Target, index < action.Target)
		return moved, doc.MoveTodoItem(index, action.Target)
	case MoveToPosition:
		moved := moveResultIndex(doc.Todos, index, action.Target, action.InsertAfter)
		return moved, doc.MoveTodoItemToPosition(index, action.Target, action.InsertAfter)
	case Indent:
		return index, doc.IndentTodoItem(index)
	case Outdent:
		return index, doc.OutdentTodoItem(index)
	case RenameHeading:
		return index, doc.RenameHeading(index, action.Text)
	case CreateHeading:
		return doc.CreateHeading(index, action.Level, action.Text)
	case SetAllChecked:
		for i := range doc.Todos {
			if doc.Todos[i].Checked != action.Checked {
				if _, err := Apply(doc, Action{Kind: SetChecked, Index: i, Checked: action.Checked}); err != nil {
					return -1, err
				}
			}
		}
		return index, nil
	case ClearDone:
		snapshot := doc.Clone()
		for i := len(doc.Todos) - 1; i >= 0; i-- {
			if doc.Todos[i].Checked {
				if _, err := Apply(doc, Action{Kind: Delete, Index: i}); err != nil {
					doc.RestoreContent(snapshot)
					return -1, err
				}
			}
		}
		return index, nil
	case SortDone, SortDue, SortPriority:
		return index, doc.SortTodos(func(a, b markdown.Todo) bool {
			switch action.Kind {
			case SortDone:
				return !a.Checked && b.Checked
			case SortDue:
				if a.DueDate == nil {
					return false
				}
				return b.DueDate == nil || a.DueDate.Before(*b.DueDate)
			default:
				if a.Priority == 0 {
					return false
				}
				return b.Priority == 0 || a.Priority < b.Priority
			}
		})
	default:
		return -1, fmt.Errorf("unknown editor action: %q", action.Kind)
	}
}

func moveResultIndex(todos []markdown.Todo, from, target int, after bool) int {
	if from == target {
		return from
	}
	if from < 0 || target < 0 || from >= len(todos) || target >= len(todos) {
		return -1
	}
	end := from + 1
	for end < len(todos) && todos[end].Depth > todos[from].Depth {
		end++
	}
	insert := target
	if after {
		insert++
		for insert < len(todos) && todos[insert].Depth > todos[target].Depth {
			insert++
		}
	}
	if insert > from {
		insert -= end - from
	}
	return insert
}
