package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestSandboxExec_WithMockRuntime(t *testing.T) {
	mock := &mockSandboxRuntime{available: true, output: "hello world\n"}
	tool := NewSandboxExecTool(mock)
	input, _ := json.Marshal(sandboxExecInput{Language: "python", Code: "print('hello world')"})
	out, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "hello world") {
		t.Errorf("output = %q, want hello world", out)
	}
}

func TestSandboxExec_NoRuntime(t *testing.T) {
	tool := NewSandboxExecTool(nil)
	input, _ := json.Marshal(sandboxExecInput{Language: "bash", Code: "echo hi"})
	out, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "sandbox not available") {
		t.Errorf("output = %q, want sandbox unavailable message", out)
	}
}

func TestSandboxExec_UnavailableRuntime(t *testing.T) {
	mock := &mockSandboxRuntime{available: false}
	tool := NewSandboxExecTool(mock)
	input, _ := json.Marshal(sandboxExecInput{Language: "bash", Code: "echo hi"})
	out, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "sandbox not available") {
		t.Errorf("output = %q, want sandbox unavailable message", out)
	}
}

func TestSandboxExec_UnsupportedLanguage(t *testing.T) {
	mock := &mockSandboxRuntime{available: true}
	tool := NewSandboxExecTool(mock)
	input, _ := json.Marshal(sandboxExecInput{Language: "java", Code: "System.out.println(1)"})
	_, err := tool.Execute(context.Background(), input)
	if err == nil {
		t.Error("expected error for unsupported language")
	}
}

func TestSandboxExec_EmptyCode(t *testing.T) {
	mock := &mockSandboxRuntime{available: true}
	tool := NewSandboxExecTool(mock)
	input, _ := json.Marshal(sandboxExecInput{Language: "python", Code: ""})
	_, err := tool.Execute(context.Background(), input)
	if err == nil {
		t.Error("expected error for empty code")
	}
}

func TestLanguageExtension(t *testing.T) {
	cases := map[string]string{
		"python": ".py",
		"node":   ".js",
		"ruby":   ".rb",
		"bash":   ".sh",
	}
	for lang, want := range cases {
		if got := languageExtension(lang); got != want {
			t.Errorf("languageExtension(%q) = %q, want %q", lang, got, want)
		}
	}
}
