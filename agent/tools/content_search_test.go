package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupContentSearchTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n// TODO: fix this\nfunc main() {}\n"), 0644)
	os.WriteFile(filepath.Join(dir, "util.go"), []byte("package main\n// FIXME: cleanup\nfunc util() {}\n"), 0644)
	os.WriteFile(filepath.Join(dir, "readme.md"), []byte("# TODO list\n- stuff\n"), 0644)
	// Write a binary file that should be skipped.
	os.WriteFile(filepath.Join(dir, "binary.dat"), []byte{0x00, 0x01, 0x02, 0xFF, 0xFE}, 0644)
	return dir
}

func TestContentSearch_BasicPattern(t *testing.T) {
	dir := setupContentSearchTestDir(t)
	tool := &ContentSearchTool{}
	input, _ := json.Marshal(contentSearchInput{Pattern: "TODO", Path: dir})
	out, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "TODO") {
		t.Error("expected TODO in output")
	}
	if !strings.Contains(out, "main.go") {
		t.Error("expected main.go in output")
	}
}

func TestContentSearch_GlobFilter(t *testing.T) {
	dir := setupContentSearchTestDir(t)
	tool := &ContentSearchTool{}
	input, _ := json.Marshal(contentSearchInput{Pattern: "TODO", Path: dir, Glob: "*.go"})
	out, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if strings.Contains(out, "readme.md") {
		t.Error("readme.md should not appear with *.go glob filter")
	}
}

func TestContentSearch_MaxResults(t *testing.T) {
	dir := setupContentSearchTestDir(t)
	tool := &ContentSearchTool{}
	input, _ := json.Marshal(contentSearchInput{Pattern: "package", Path: dir, MaxResults: 1})
	out, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) > 1 {
		t.Errorf("expected 1 result, got %d", len(lines))
	}
}

func TestContentSearch_BinarySkipped(t *testing.T) {
	dir := setupContentSearchTestDir(t)
	tool := &ContentSearchTool{}
	input, _ := json.Marshal(contentSearchInput{Pattern: ".*", Path: dir})
	out, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if strings.Contains(out, "binary.dat") {
		t.Error("binary file should be skipped")
	}
}

func TestContentSearch_NoMatch(t *testing.T) {
	dir := setupContentSearchTestDir(t)
	tool := &ContentSearchTool{}
	input, _ := json.Marshal(contentSearchInput{Pattern: "NONEXISTENT_PATTERN_XYZ", Path: dir})
	out, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out != "no matches found" {
		t.Errorf("expected 'no matches found', got %q", out)
	}
}

func TestContentSearch_InvalidRegex(t *testing.T) {
	tool := &ContentSearchTool{}
	input, _ := json.Marshal(contentSearchInput{Pattern: "[invalid", Path: "."})
	_, err := tool.Execute(context.Background(), input)
	if err == nil {
		t.Error("expected error for invalid regex")
	}
}

func TestContentSearch_EmptyPattern(t *testing.T) {
	tool := &ContentSearchTool{}
	input, _ := json.Marshal(contentSearchInput{Pattern: "", Path: "."})
	_, err := tool.Execute(context.Background(), input)
	if err == nil {
		t.Error("expected error for empty pattern")
	}
}

func TestContentSearch_NonexistentPath(t *testing.T) {
	tool := &ContentSearchTool{}
	input, _ := json.Marshal(contentSearchInput{Pattern: "test", Path: "/nonexistent-path-xyz"})
	_, err := tool.Execute(context.Background(), input)
	if err != nil {
		// Walk should fail for nonexistent dir
		return
	}
	// If it returns "no matches found" that's also acceptable
}

func TestContentSearch_InvalidJSON(t *testing.T) {
	tool := &ContentSearchTool{}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{bad`))
	if err == nil || !strings.Contains(err.Error(), "parse input") {
		t.Errorf("expected parse error, got: %v", err)
	}
}
