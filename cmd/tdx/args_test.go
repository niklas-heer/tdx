package main

import (
	"reflect"
	"testing"
)

func TestParseArgs(t *testing.T) {
	defaults := DefaultConfig().Defaults
	for _, tt := range []struct {
		name          string
		args          []string
		file, command string
		text          []string
	}{
		{"default file", []string{"list"}, defaults.File, "list", nil},
		{"positional path", []string{"my tasks.md", "list"}, "my tasks.md", "list", nil},
		{"explicit path", []string{"-f", "TASKS", "list"}, "TASKS", "list", nil},
		{"equals and after command", []string{"list", "--file=TASKS"}, "TASKS", "list", nil},
		{"literal flags", []string{"add", "--", "--read-only", "--json"}, defaults.File, "add", []string{"--read-only", "--json"}},
		{"quoted text", []string{"add", `"Käse"`}, defaults.File, "add", []string{`"Käse"`}},
		{"literal filename", []string{"--", "-tasks.md", "list"}, "-tasks.md", "list", nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			opts, err := parseArgs(tt.args, defaults)
			if err != nil {
				t.Fatal(err)
			}
			if opts.File != tt.file || opts.Command != tt.command || !reflect.DeepEqual(opts.Args, tt.text) {
				t.Fatalf("unexpected options: %+v", opts)
			}
		})
	}
	for _, args := range [][]string{
		{"--file"}, {"--file="}, {"one.md", "--file=two.md"},
		{"list", "--status=invalid"}, {"list", "--tag="}, {"list", "--tag=#"},
		{"add", "--json", "task"}, {"list", "--json=false"}, {"--unknown"},
		{"--max-visible=-1"}, {"--max-visible=abc"}, {"--max-visible"},
	} {
		if _, err := parseArgs(args, defaults); err == nil {
			t.Errorf("accepted invalid args: %q", args)
		}
	}
	opts, err := parseArgs([]string{"list", "--json", "--status=open", "--tag=#backend", "--tag", "urgent", "--read-only"}, defaults)
	if err != nil || !opts.List.JSON || opts.List.Status != "open" || !opts.ReadOnly || !reflect.DeepEqual(opts.List.Tags, []string{"backend", "urgent"}) {
		t.Fatalf("filtered options = %+v, err = %v", opts, err)
	}
}
