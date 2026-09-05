// go-probe compares a minimal parser/patch path and the production editor path.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/niklas-heer/tdx/internal/editor"
	"github.com/niklas-heer/tdx/internal/markdown"
	"github.com/yuin/goldmark/ast"
	extast "github.com/yuin/goldmark/extension/ast"
)

type marker struct {
	Checked bool `json:"checked"`
	Depth   int  `json:"depth"`
	offset  int
}

func scan(source string) []marker {
	_, body, _ := markdown.ParseMetadata(source)
	base := len(source) - len(body)
	doc, _ := markdown.ParseAST(body)
	markers := []marker{}
	depth := 0
	_ = ast.Walk(doc.AST, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if node.Kind() == ast.KindList {
			if entering {
				depth++
			} else {
				depth--
			}
		}
		if check, ok := node.(*extast.TaskCheckBox); ok && entering {
			offset := -1
			parent := node.Parent()
			if parent.Lines().Len() > 0 {
				line := parent.Lines().At(0)
				start := line.Start
				if relative := strings.Index(body[start:line.Stop], "["); relative >= 0 {
					offset = base + start + relative
				}
			}
			markers = append(markers, marker{Checked: check.IsChecked, Depth: max(0, depth-1), offset: offset})
		}
		return ast.WalkContinue, nil
	})
	return markers
}

func patch(source string, index int) (string, error) {
	markers := scan(source)
	if index < 0 || index >= len(markers) {
		return "", fmt.Errorf("invalid task index")
	}
	m := markers[index]
	start := m.offset
	if start < 0 || start+2 >= len(source) || source[start] != '[' || source[start+2] != ']' {
		return "", fmt.Errorf("parser marker range is not a checkbox")
	}
	value := "x"
	if m.Checked {
		value = " "
	}
	return source[:start+1] + value + source[start+2:], nil
}

func production(source string, index int) (string, error) {
	meta, body, _ := markdown.ParseMetadata(source)
	doc := markdown.ParseMarkdown(body)
	doc.Metadata = meta
	if _, err := editor.Apply(doc, editor.Action{Kind: editor.Toggle, Index: index}); err != nil {
		return "", err
	}
	return markdown.SerializeMarkdown(doc), nil
}

var sink any

func run() error {
	if len(os.Args) != 4 {
		return fmt.Errorf("usage: go-probe inspect|patch|production|scan|patch-bench|production-bench FILE INDEX_OR_ITERATIONS")
	}
	source, err := os.ReadFile(os.Args[2])
	if err != nil {
		return err
	}
	n, err := strconv.Atoi(os.Args[3])
	if err != nil || n < 0 {
		return fmt.Errorf("expected a non-negative integer")
	}
	text := string(source)
	switch os.Args[1] {
	case "inspect":
		return json.NewEncoder(os.Stdout).Encode(scan(text))
	case "production-inspect":
		_, body, _ := markdown.ParseMetadata(text)
		doc := markdown.ParseMarkdown(body)
		markers := []marker{}
		for _, todo := range doc.Todos {
			markers = append(markers, marker{Checked: todo.Checked, Depth: todo.Depth})
		}
		return json.NewEncoder(os.Stdout).Encode(markers)
	case "patch", "production":
		operation := patch
		if os.Args[1] == "production" {
			operation = production
		}
		out, err := operation(text, n)
		if err != nil {
			return err
		}
		_, err = fmt.Print(out)
		return err
	case "scan", "patch-bench", "production-bench":
		if n == 0 {
			return fmt.Errorf("iterations must be positive")
		}
		started := time.Now()
		checksum := 0
		for range n {
			if os.Args[1] == "scan" {
				result := scan(text)
				sink = result
				checksum += len(result)
			} else {
				operation := patch
				if os.Args[1] == "production-bench" {
					operation = production
				}
				result, err := operation(text, 0)
				if err != nil {
					return err
				}
				sink = result
				checksum += len(result)
			}
		}
		runtime.KeepAlive(sink)
		return json.NewEncoder(os.Stdout).Encode(map[string]any{"ns_per_op": float64(time.Since(started).Nanoseconds()) / float64(n), "checksum": checksum})
	default:
		return fmt.Errorf("unknown operation")
	}
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
