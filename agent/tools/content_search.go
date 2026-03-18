package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ContentSearchTool searches file contents using regex. Pure Go, no shell.
type ContentSearchTool struct{}

func (c *ContentSearchTool) Name() string        { return "content_search" }
func (c *ContentSearchTool) Description() string { return "Search file contents by regex pattern" }
func (c *ContentSearchTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"pattern": {"type": "string", "description": "Regex pattern to search for"},
			"path": {"type": "string", "description": "Directory to search in"},
			"glob": {"type": "string", "description": "Glob filter for file names (e.g. *.go)"},
			"max_results": {"type": "integer", "description": "Max matches to return (default 50)"}
		},
		"required": ["pattern", "path"]
	}`)
}

type contentSearchInput struct {
	Pattern    string `json:"pattern"`
	Path       string `json:"path"`
	Glob       string `json:"glob"`
	MaxResults int    `json:"max_results"`
}

func (c *ContentSearchTool) Execute(_ context.Context, input json.RawMessage) (string, error) {
	var in contentSearchInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("parse input: %w", err)
	}
	if in.Pattern == "" {
		return "", fmt.Errorf("pattern is required")
	}
	if in.MaxResults <= 0 {
		in.MaxResults = 50
	}

	re, err := regexp.Compile(in.Pattern)
	if err != nil {
		return "", fmt.Errorf("invalid regex: %w", err)
	}

	var b strings.Builder
	count := 0

	walkErr := filepath.WalkDir(in.Path, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if in.Glob != "" {
			matched, _ := filepath.Match(in.Glob, d.Name())
			if !matched {
				return nil
			}
		}
		if isBinaryFile(path) {
			return nil
		}
		matches, scanErr := searchFile(path, re, in.MaxResults-count)
		if scanErr != nil {
			return nil
		}
		for _, m := range matches {
			fmt.Fprintf(&b, "%s:%d: %s\n", path, m.line, m.text)
			count++
			if count >= in.MaxResults {
				return fs.SkipAll
			}
		}
		return nil
	})
	if walkErr != nil {
		return "", fmt.Errorf("walk: %w", walkErr)
	}

	result := b.String()
	if result == "" {
		return "no matches found", nil
	}
	return TruncateHead(result, MaxLinesFileRead), nil
}

type searchMatch struct {
	line int
	text string
}

func searchFile(path string, re *regexp.Regexp, maxMatches int) ([]searchMatch, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var matches []searchMatch
	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		text := scanner.Text()
		if re.MatchString(text) {
			if len(text) > 200 {
				text = text[:200] + "..."
			}
			matches = append(matches, searchMatch{line: lineNum, text: text})
			if len(matches) >= maxMatches {
				break
			}
		}
	}
	return matches, scanner.Err()
}

func isBinaryFile(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return true
	}
	defer f.Close()

	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil || n == 0 {
		return true
	}
	ct := http.DetectContentType(buf[:n])
	return !strings.HasPrefix(ct, "text/")
}
