// Package pipeline contains pure configuration logic for the Dagger pipeline.
package pipeline

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
)

type Metadata struct {
	Version     string
	Description string
}

type Target struct {
	Name string
	OS   string
	Arch string
}

var releaseTargets = []Target{
	{Name: "tdx-darwin-amd64", OS: "darwin", Arch: "amd64"},
	{Name: "tdx-darwin-arm64", OS: "darwin", Arch: "arm64"},
	{Name: "tdx-linux-amd64", OS: "linux", Arch: "amd64"},
	{Name: "tdx-linux-arm64", OS: "linux", Arch: "arm64"},
	{Name: "tdx-windows-amd64.exe", OS: "windows", Arch: "amd64"},
}

func ReleaseTargets() []Target {
	return append([]Target(nil), releaseTargets...)
}

func ParseMetadata(contents string) (Metadata, error) {
	var metadata Metadata
	scanner := bufio.NewScanner(strings.NewReader(contents))
	for scanner.Scan() {
		key, value, ok := strings.Cut(scanner.Text(), "=")
		if !ok {
			continue
		}
		value = strings.Trim(strings.TrimSpace(value), `"`)
		switch strings.TrimSpace(key) {
		case "version":
			metadata.Version = value
		case "description":
			metadata.Description = value
		}
	}
	if err := scanner.Err(); err != nil {
		return Metadata{}, fmt.Errorf("parse tdx.toml: %w", err)
	}
	if metadata.Version == "" || metadata.Description == "" {
		return Metadata{}, fmt.Errorf("tdx.toml must define version and description")
	}
	return metadata, nil
}

func CoverageColor(coverage float64) string {
	switch {
	case coverage >= 80:
		return "brightgreen"
	case coverage >= 60:
		return "green"
	case coverage >= 40:
		return "yellow"
	default:
		return "red"
	}
}

func ParseTotalCoverage(report string) (string, error) {
	for _, line := range strings.Split(report, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 || fields[0] != "total:" {
			continue
		}
		percent := strings.TrimSuffix(fields[len(fields)-1], "%")
		if _, err := strconv.ParseFloat(percent, 64); err != nil {
			return "", fmt.Errorf("parse total coverage %q: %w", percent, err)
		}
		return percent, nil
	}
	return "", fmt.Errorf("coverage report has no total")
}

func CoverageBadgeJSON(percent string) (string, error) {
	coverage, err := strconv.ParseFloat(percent, 64)
	if err != nil {
		return "", fmt.Errorf("parse badge coverage %q: %w", percent, err)
	}
	return fmt.Sprintf(
		"{\"schemaVersion\":1,\"label\":\"coverage\",\"message\":\"%s%%\",\"color\":\"%s\"}\n",
		percent,
		CoverageColor(coverage),
	), nil
}
