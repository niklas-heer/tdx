// Command go-history exposes the production version store to interoperability tests.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/niklas-heer/tdx/internal/versioning"
)

func run() (err error) {
	if len(os.Args) < 3 {
		return fmt.Errorf("usage: go-history list|read|capture FILE [ID]")
	}
	path, err := filepath.Abs(os.Args[2])
	if err != nil {
		return err
	}
	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return err
	}
	store, err := versioning.Open(0)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, store.Close()) }()
	switch os.Args[1] {
	case "list":
		versions, e := store.ListVersions(path)
		if e != nil {
			return e
		}
		return json.NewEncoder(os.Stdout).Encode(versions)
	case "read":
		if len(os.Args) != 4 {
			return fmt.Errorf("read requires ID")
		}
		id, e := strconv.ParseInt(os.Args[3], 10, 64)
		if e != nil {
			return e
		}
		content, e := store.ReadVersion(path, id)
		if e != nil {
			return e
		}
		_, err = fmt.Fprint(os.Stdout, content)
		return err
	case "capture":
		content, e := os.ReadFile(path)
		if e != nil {
			return e
		}
		return store.SaveVersion(path, string(content))
	default:
		return fmt.Errorf("unsupported operation %q", os.Args[1])
	}
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
