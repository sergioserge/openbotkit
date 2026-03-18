package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileWriteTool_NoInteractor(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	tool := NewFileWriteTool(nil, nil)
	input, _ := json.Marshal(map[string]string{"path": path, "content": "hello"})
	out, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "wrote 5 bytes") {
		t.Errorf("output = %q", out)
	}
	got, _ := os.ReadFile(path)
	if string(got) != "hello" {
		t.Errorf("file content = %q, want hello", got)
	}
}

func TestFileWriteTool_WithInteractor_Approved(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	inter := &mockInteractor{approveAll: true}
	tool := NewFileWriteTool(inter, nil)
	input, _ := json.Marshal(map[string]string{"path": path, "content": "approved"})
	out, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "wrote 8 bytes") {
		t.Errorf("output = %q", out)
	}
}

func TestFileWriteTool_WithInteractor_Denied(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	inter := &mockInteractor{approveAll: false}
	tool := NewFileWriteTool(inter, nil)
	input, _ := json.Marshal(map[string]string{"path": path, "content": "denied"})
	out, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out != "denied_by_user" {
		t.Errorf("output = %q, want denied_by_user", out)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("file should not exist after denial")
	}
}

func TestFileEditTool_NoInteractor(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("hello world"), 0644)
	tool := NewFileEditTool(nil, nil)
	input, _ := json.Marshal(map[string]string{
		"path": path, "old_string": "world", "new_string": "go",
	})
	out, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "replaced 1 occurrence") {
		t.Errorf("output = %q", out)
	}
	got, _ := os.ReadFile(path)
	if string(got) != "hello go" {
		t.Errorf("file content = %q, want 'hello go'", got)
	}
}

func TestFileEditTool_WithInteractor_Denied(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("hello world"), 0644)
	inter := &mockInteractor{approveAll: false}
	tool := NewFileEditTool(inter, nil)
	input, _ := json.Marshal(map[string]string{
		"path": path, "old_string": "world", "new_string": "go",
	})
	out, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out != "denied_by_user" {
		t.Errorf("output = %q, want denied_by_user", out)
	}
	got, _ := os.ReadFile(path)
	if string(got) != "hello world" {
		t.Error("file should be unchanged after denial")
	}
}
