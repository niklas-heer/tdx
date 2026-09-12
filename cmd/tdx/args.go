package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/niklas-heer/tdx/internal/cmd"
	"github.com/niklas-heer/tdx/internal/markdown"
)

type cliOptions struct {
	File             string
	Command          string
	Args             []string
	ReadOnly         bool
	ShowHeadings     bool
	MaxVisible       int
	List             cmd.ListOptions
	ExpectedRevision string
}

// parseArgs keeps option parsing separate from configuration, storage, and UI setup.
// A -- delimiter makes all following arguments literal, including known flags.
func parseArgs(args []string, defaults DefaultsConfig) (cliOptions, error) {
	opts := cliOptions{File: defaults.File, ReadOnly: defaults.ReadOnly, ShowHeadings: defaults.ShowHeadings, MaxVisible: -1}
	literal, fileSet, listFlags := false, false, false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !literal && arg == "--" {
			literal = true
			continue
		}
		if !literal && strings.HasPrefix(arg, "-") {
			name, value, hasValue := strings.Cut(arg, "=")
			takeValue := func() (string, error) {
				if hasValue {
					return value, nil
				}
				if i+1 == len(args) {
					return "", fmt.Errorf("%s requires a value", name)
				}
				i++
				return args[i], nil
			}
			switch name {
			case "--file", "-f", "--max-visible", "-m", "--status", "--tag", "--due", "--priority", "--section", "--if-revision":
				value, err := takeValue()
				if err != nil {
					return opts, err
				}
				switch name {
				case "--file", "-f":
					if value == "" || fileSet {
						return opts, fmt.Errorf("specify exactly one non-empty file path")
					}
					opts.File, fileSet = value, true
				case "--max-visible", "-m":
					n, err := strconv.Atoi(value)
					if err != nil || n < 0 {
						return opts, fmt.Errorf("--max-visible requires a non-negative integer")
					}
					opts.MaxVisible = n
				case "--status":
					if value != "all" && value != "open" && value != "done" {
						return opts, fmt.Errorf("--status must be all, open, or done")
					}
					opts.List.Status, listFlags = value, true
				case "--due":
					opts.List.Due, listFlags = value, true
					if value == "" {
						return opts, fmt.Errorf("--due requires a value")
					}
				case "--priority":
					n, err := strconv.Atoi(value)
					if err != nil || n < 0 {
						return opts, fmt.Errorf("--priority requires a non-negative integer")
					}
					opts.List.Priority, listFlags = &n, true
				case "--section":
					if strings.TrimSpace(value) == "" {
						return opts, fmt.Errorf("--section requires a heading title")
					}
					opts.List.Section, listFlags = value, true
				case "--if-revision":
					if err := markdown.ValidateRevisionToken(value); err != nil {
						return opts, err
					}
					opts.ExpectedRevision = value
				case "--tag":
					value = strings.TrimPrefix(value, "#")
					if value == "" || strings.ContainsAny(value, " \t\r\n") {
						return opts, fmt.Errorf("--tag requires a non-empty tag")
					}
					opts.List.Tags = append(opts.List.Tags, value)
					listFlags = true
				}
			case "--read-only", "-r", "--manual-save", "--show-headings", "--json", "--with-revision":
				if hasValue {
					return opts, fmt.Errorf("%s does not take a value", name)
				}
				switch name {
				case "--read-only", "-r", "--manual-save":
					opts.ReadOnly = true
				case "--show-headings":
					opts.ShowHeadings = true
				case "--with-revision":
					opts.List.WithRevision, listFlags = true, true
				case "--json":
					opts.List.JSON, listFlags = true, true
				}
			case "--help", "-h":
				if hasValue {
					return opts, fmt.Errorf("%s does not take a value", name)
				}
				opts.Command, opts.Args = "help", nil
				return opts, nil
			case "--version", "-v", "--debug-config":
				if hasValue || opts.Command != "" {
					return opts, fmt.Errorf("use tdx %s on its own", name)
				}
				opts.Command = name
			default:
				return opts, fmt.Errorf("unknown option %s (use -- before literal task text)", name)
			}
			continue
		}
		if opts.Command == "" {
			if !fileSet && strings.HasSuffix(arg, ".md") {
				opts.File, fileSet = arg, true
			} else {
				opts.Command = arg
			}
		} else {
			opts.Args = append(opts.Args, arg)
		}
	}
	if listFlags && opts.Command != "list" {
		return opts, fmt.Errorf("query options are only supported by list")
	}
	if err := opts.List.Validate(); err != nil {
		return opts, err
	}
	if opts.ExpectedRevision != "" {
		switch opts.Command {
		case "add", "edit", "toggle", "delete", "done", "undone":
		default:
			return opts, fmt.Errorf("--if-revision is only supported by task mutations")
		}
	}
	return opts, nil
}
