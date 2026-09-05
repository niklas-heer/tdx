package editor

import (
	"github.com/niklas-heer/tdx/internal/markdown"
	"sort"
)

// Section represents a group of todos under a heading (or before any heading)
type Section struct {
	Start int // Index of first todo in this section
	End   int // Index after last todo in this section (exclusive)
}

// Sections divides todos into sections based on headings
// Each section contains todos that belong under a particular heading
func Sections(todos []markdown.Todo, headings []markdown.Heading) []Section {
	if len(todos) == 0 {
		return nil
	}

	// Sort headings by BeforeTodoIndex to process in order
	sortedHeadings := make([]markdown.Heading, len(headings))
	copy(sortedHeadings, headings)
	sort.Slice(sortedHeadings, func(i, j int) bool {
		return sortedHeadings[i].BeforeTodoIndex < sortedHeadings[j].BeforeTodoIndex
	})

	var sections []Section

	// Find section boundaries from headings
	prevBoundary := 0
	for _, h := range sortedHeadings {
		// BeforeTodoIndex tells us which todo this heading appears before
		if h.BeforeTodoIndex > prevBoundary && h.BeforeTodoIndex <= len(todos) {
			// There are todos before this heading that form a section
			sections = append(sections, Section{
				Start: prevBoundary,
				End:   h.BeforeTodoIndex,
			})
			prevBoundary = h.BeforeTodoIndex
		}
	}

	// Add final section for remaining todos
	if prevBoundary < len(todos) {
		sections = append(sections, Section{
			Start: prevBoundary,
			End:   len(todos),
		})
	}

	// If no sections were created (no headings), treat all todos as one section
	if len(sections) == 0 {
		sections = append(sections, Section{
			Start: 0,
			End:   len(todos),
		})
	}

	return sections
}

// SortTodosInSections sorts todos within each section using the provided sort function
// sortFn should sort the slice in place
func SortTodosInSections(todos []markdown.Todo, headings []markdown.Heading, sortFn func([]markdown.Todo)) {
	sections := Sections(todos, headings)

	for _, section := range sections {
		if section.End > section.Start {
			sectionTodos := todos[section.Start:section.End]
			sortFn(sectionTodos)
		}
	}
}
