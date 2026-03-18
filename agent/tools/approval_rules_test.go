package tools

import (
	"encoding/json"
	"testing"
	"time"
)

func TestApprovalRuleSet_ManualRule(t *testing.T) {
	s := NewApprovalRuleSet()
	s.Add(ApprovalRule{ToolName: "slack_send", Pattern: "#general"})

	input, _ := json.Marshal(map[string]string{"channel": "#general"})
	if !s.Matches("slack_send", input) {
		t.Error("expected match for manual rule")
	}

	otherInput, _ := json.Marshal(map[string]string{"channel": "#random"})
	if s.Matches("slack_send", otherInput) {
		t.Error("should not match different channel")
	}
}

func TestApprovalRuleSet_AutoGenerateAfterThreshold(t *testing.T) {
	s := NewApprovalRuleSet()
	input, _ := json.Marshal(map[string]string{"channel": "#general"})

	// Record approvals below threshold.
	for i := 0; i < autoApproveThreshold-1; i++ {
		s.RecordApproval("slack_send", input)
	}
	if s.Matches("slack_send", input) {
		t.Error("should not auto-approve below threshold")
	}

	// One more pushes it over.
	s.RecordApproval("slack_send", input)
	if !s.Matches("slack_send", input) {
		t.Error("should auto-approve after threshold")
	}
}

func TestApprovalRuleSet_DifferentPatternsTrackedSeparately(t *testing.T) {
	s := NewApprovalRuleSet()
	general, _ := json.Marshal(map[string]string{"channel": "#general"})
	random, _ := json.Marshal(map[string]string{"channel": "#random"})

	for i := 0; i < autoApproveThreshold; i++ {
		s.RecordApproval("slack_send", general)
	}
	if !s.Matches("slack_send", general) {
		t.Error("should auto-approve #general")
	}
	if s.Matches("slack_send", random) {
		t.Error("should not auto-approve #random")
	}
}

func TestApprovalRuleSet_ExpiredRule(t *testing.T) {
	s := NewApprovalRuleSet()
	s.Add(ApprovalRule{
		ToolName:  "bash",
		ExpiresAt: time.Now().Add(-1 * time.Minute),
	})
	input, _ := json.Marshal(map[string]string{"command": "echo hi"})
	if s.Matches("bash", input) {
		t.Error("expired rule should not match")
	}
}

func TestApprovalRuleSet_IsRubberStamping(t *testing.T) {
	s := NewApprovalRuleSet()
	input, _ := json.Marshal(map[string]string{"channel": "#general"})

	for i := 0; i < rubberStampThreshold-1; i++ {
		s.RecordApproval("slack_send", input)
	}
	if s.IsRubberStamping() {
		t.Error("should not flag as rubber-stamping below threshold")
	}

	s.RecordApproval("slack_send", input)
	if !s.IsRubberStamping() {
		t.Error("should flag as rubber-stamping at threshold")
	}
}

func TestApprovalRuleSet_HistoryBounded(t *testing.T) {
	s := NewApprovalRuleSet()
	input, _ := json.Marshal(map[string]string{"channel": "#test"})
	for i := 0; i < maxHistoryLen+50; i++ {
		s.RecordApproval("slack_send", input)
	}
	s.mu.Lock()
	n := len(s.history)
	s.mu.Unlock()
	if n > maxHistoryLen {
		t.Errorf("history len = %d, want <= %d", n, maxHistoryLen)
	}
}

func TestExtractPattern_SlackChannel(t *testing.T) {
	input, _ := json.Marshal(map[string]string{"channel": "#general"})
	if p := extractPattern("slack_send", input); p != "#general" {
		t.Errorf("pattern = %q, want #general", p)
	}
}

func TestExtractPattern_GWSCommand(t *testing.T) {
	input, _ := json.Marshal(map[string]string{"command": "calendar events.list --maxResults 10"})
	if p := extractPattern("gws_execute", input); p != "calendar" {
		t.Errorf("pattern = %q, want calendar", p)
	}
}

func TestExtractPattern_UnknownTool(t *testing.T) {
	input, _ := json.Marshal(map[string]string{"foo": "bar"})
	if p := extractPattern("unknown", input); p != "unknown" {
		t.Errorf("pattern = %q, want unknown", p)
	}
}

func TestExtractPattern_InvalidJSON(t *testing.T) {
	if p := extractPattern("slack_send", json.RawMessage(`{bad`)); p != "" {
		t.Errorf("pattern = %q, want empty for invalid JSON", p)
	}
}

func TestExtractPattern_GWSSingleWordCommand(t *testing.T) {
	input, _ := json.Marshal(map[string]string{"command": "calendar"})
	if p := extractPattern("gws_execute", input); p != "calendar" {
		t.Errorf("pattern = %q, want calendar", p)
	}
}

func TestExtractPattern_SlackMissingChannel(t *testing.T) {
	input, _ := json.Marshal(map[string]string{"text": "hello"})
	if p := extractPattern("slack_send", input); p != "slack_send" {
		t.Errorf("pattern = %q, want slack_send (fallback)", p)
	}
}

func TestExtractPattern_GWSMissingCommand(t *testing.T) {
	input, _ := json.Marshal(map[string]string{"foo": "bar"})
	if p := extractPattern("gws_execute", input); p != "gws_execute" {
		t.Errorf("pattern = %q, want gws_execute (fallback)", p)
	}
}

func TestExtractPattern_BashCommand(t *testing.T) {
	input, _ := json.Marshal(map[string]string{"command": "curl example.com"})
	if p := extractPattern("bash", input); p != "curl" {
		t.Errorf("pattern = %q, want curl", p)
	}
}

func TestExtractPattern_BashSingleWord(t *testing.T) {
	input, _ := json.Marshal(map[string]string{"command": "ls"})
	if p := extractPattern("bash", input); p != "ls" {
		t.Errorf("pattern = %q, want ls", p)
	}
}

func TestExtractPattern_FileWrite(t *testing.T) {
	input, _ := json.Marshal(map[string]string{"path": "/tmp/test.txt", "content": "hello"})
	if p := extractPattern("file_write", input); p != "/tmp/test.txt" {
		t.Errorf("pattern = %q, want /tmp/test.txt", p)
	}
}

func TestExtractPattern_FileEdit(t *testing.T) {
	input, _ := json.Marshal(map[string]string{"path": "/tmp/test.txt", "old_string": "a", "new_string": "b"})
	if p := extractPattern("file_edit", input); p != "/tmp/test.txt" {
		t.Errorf("pattern = %q, want /tmp/test.txt", p)
	}
}

func TestApprovalRuleSet_WildcardPattern(t *testing.T) {
	s := NewApprovalRuleSet()
	s.Add(ApprovalRule{ToolName: "bash", Pattern: ""})
	input, _ := json.Marshal(map[string]string{"command": "anything"})
	if !s.Matches("bash", input) {
		t.Error("empty pattern should match any input")
	}
}

func TestApprovalRuleSet_DuplicateRulePrevention(t *testing.T) {
	s := NewApprovalRuleSet()
	input, _ := json.Marshal(map[string]string{"channel": "#general"})
	for i := 0; i < autoApproveThreshold*3; i++ {
		s.RecordApproval("slack_send", input)
	}
	s.mu.Lock()
	count := 0
	for _, r := range s.rules {
		if r.ToolName == "slack_send" && r.Pattern == "#general" {
			count++
		}
	}
	s.mu.Unlock()
	if count != 1 {
		t.Errorf("expected 1 rule, got %d (duplicate prevention failed)", count)
	}
}
