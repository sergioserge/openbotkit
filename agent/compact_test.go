package agent

import (
	"fmt"
	"strings"
	"testing"

	"github.com/priyanshujain/openbotkit/provider"
)

func TestCompactHistory_BelowThreshold(t *testing.T) {
	a := &Agent{maxHistory: 40}
	for i := range 10 {
		a.history = append(a.history, provider.NewTextMessage(
			provider.RoleUser, fmt.Sprintf("msg %d", i)))
	}
	a.compactHistory()
	if len(a.history) != 10 {
		t.Errorf("history len = %d, want 10 (no compaction)", len(a.history))
	}
}

func TestCompactHistory_AboveThreshold(t *testing.T) {
	a := &Agent{maxHistory: 40}
	for i := range 50 {
		a.history = append(a.history, provider.NewTextMessage(
			provider.RoleUser, fmt.Sprintf("msg %d", i)))
	}
	a.compactHistory()

	// Should be summary + keepMessages = 21
	if len(a.history) != 21 {
		t.Fatalf("history len = %d, want 21", len(a.history))
	}

	// First message should be the summary.
	summary := a.history[0].Content[0].Text
	if !strings.Contains(summary, "30 messages removed") {
		t.Errorf("summary = %q, expected '30 messages removed'", summary)
	}

	// Last message should be the original last message.
	last := a.history[20].Content[0].Text
	if last != "msg 49" {
		t.Errorf("last message = %q, want 'msg 49'", last)
	}
}

func TestCompactHistory_ExactThreshold(t *testing.T) {
	a := &Agent{maxHistory: 40}
	for i := range 40 {
		a.history = append(a.history, provider.NewTextMessage(
			provider.RoleUser, fmt.Sprintf("msg %d", i)))
	}
	a.compactHistory()
	if len(a.history) != 40 {
		t.Errorf("history len = %d, want 40 (at threshold, no compaction)", len(a.history))
	}
}
