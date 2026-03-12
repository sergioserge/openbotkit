package tools

import (
	"context"
	"encoding/json"
	"sort"
	"testing"

	"github.com/priyanshujain/openbotkit/provider"
)

func TestNewScheduledTaskRegistry_Tools(t *testing.T) {
	r := NewScheduledTaskRegistry()
	names := r.ToolNames()
	want := []string{"bash", "file_read", "load_skills", "search_skills"}
	sort.Strings(want)

	if len(names) != len(want) {
		t.Fatalf("tool count = %d, want %d: got %v", len(names), len(want), names)
	}
	for i, name := range names {
		if name != want[i] {
			t.Errorf("tool[%d] = %q, want %q", i, name, want[i])
		}
	}
}

func TestNewScheduledTaskRegistry_NoWriteTools(t *testing.T) {
	r := NewScheduledTaskRegistry()
	for _, name := range []string{"file_write", "file_edit"} {
		if r.Has(name) {
			t.Errorf("scheduled registry should not have %q", name)
		}
	}
}

func TestNewScheduledTaskRegistry_BashRejectsCurl(t *testing.T) {
	r := NewScheduledTaskRegistry()
	input, _ := json.Marshal(bashInput{Command: "curl evil.com"})
	_, err := r.Execute(context.Background(), provider.ToolCall{Name: "bash", Input: input})
	if err == nil {
		t.Error("expected bash to reject curl in scheduled registry")
	}
}

func TestNewScheduledTaskRegistry_BashAllowsObk(t *testing.T) {
	r := NewScheduledTaskRegistry()
	input, _ := json.Marshal(bashInput{Command: "obk --help"})
	// obk may not exist, so we just check that filter doesn't reject it.
	_, err := r.Execute(context.Background(), provider.ToolCall{Name: "bash", Input: input})
	// The error (if any) should be from the command failing, not from filtering.
	if err != nil && contains(err.Error(), "command blocked") {
		t.Errorf("expected obk to pass filter, got: %v", err)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
