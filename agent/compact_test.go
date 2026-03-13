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

	// keep = maxHistory/2 = 20, so result is summary + 20 = 21
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

func TestCompactHistory_SmallMaxHistory(t *testing.T) {
	// maxHistory=6, keep=6/2=3, so 11 messages becomes summary + 3 = 4.
	a := &Agent{maxHistory: 6}
	for i := range 11 {
		a.history = append(a.history, provider.NewTextMessage(
			provider.RoleUser, fmt.Sprintf("msg %d", i)))
	}
	a.compactHistory()

	if len(a.history) != 4 {
		t.Errorf("history len = %d, want 4 (summary + 3 kept)", len(a.history))
	}

	// Last message should be the original last message.
	last := a.history[3].Content[0].Text
	if last != "msg 10" {
		t.Errorf("last message = %q, want 'msg 10'", last)
	}
}

func TestCompactHistory_EmptyHistory(t *testing.T) {
	a := &Agent{maxHistory: 40}
	a.compactHistory() // should not panic
	if len(a.history) != 0 {
		t.Errorf("history len = %d, want 0", len(a.history))
	}
}

func TestCompactHistory_MaxHistoryOne(t *testing.T) {
	// maxHistory=1, keep=1/2=0, guard sets keep=1.
	// 5 messages → summary + 1 kept = 2.
	a := &Agent{maxHistory: 1}
	for i := range 5 {
		a.history = append(a.history, provider.NewTextMessage(
			provider.RoleUser, fmt.Sprintf("msg %d", i)))
	}
	a.compactHistory()

	if len(a.history) != 2 {
		t.Fatalf("history len = %d, want 2 (summary + 1 kept)", len(a.history))
	}

	last := a.history[1].Content[0].Text
	if last != "msg 4" {
		t.Errorf("last message = %q, want 'msg 4'", last)
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

// --- Microcompaction tests ---

func toolResultMsg(name, content string) provider.Message {
	return provider.Message{
		Role: provider.RoleUser,
		Content: []provider.ContentBlock{{
			Type: provider.ContentToolResult,
			ToolResult: &provider.ToolResult{
				ToolUseID: "id-" + name,
				Name:      name,
				Content:   content,
			},
		}},
	}
}

func TestMicrocompact_RecentPreserved(t *testing.T) {
	a := &Agent{maxHistory: 40}
	bigContent := strings.Repeat("x", 300)
	// 8 messages: 4 pairs of (assistant, tool_result)
	for i := range 4 {
		a.history = append(a.history,
			provider.NewTextMessage(provider.RoleAssistant, fmt.Sprintf("resp %d", i)),
			toolResultMsg("bash", bigContent),
		)
	}
	a.microcompact()
	// With 8 messages, cutoff = 8 - 6 = 2, so first 2 messages are old.
	// Messages at index 2-7 are recent and should be preserved.
	for i := 2; i < len(a.history); i++ {
		msg := a.history[i]
		if msg.Role == provider.RoleUser && msg.Content[0].ToolResult != nil {
			if strings.HasPrefix(msg.Content[0].ToolResult.Content, "[Previous:") {
				t.Errorf("recent message at index %d was compacted", i)
			}
		}
	}
}

func TestMicrocompact_OldResultsCompacted(t *testing.T) {
	a := &Agent{maxHistory: 40}
	bigContent := strings.Repeat("x", 300)
	// 10 messages: 5 pairs
	for i := range 5 {
		a.history = append(a.history,
			provider.NewTextMessage(provider.RoleAssistant, fmt.Sprintf("resp %d", i)),
			toolResultMsg("bash", bigContent),
		)
	}
	a.microcompact()
	// cutoff = 10 - 6 = 4. Messages 0-3 are old.
	// Message 1 is a tool result (index 1 in the loop).
	found := false
	for i := 0; i < 4; i++ {
		msg := a.history[i]
		if msg.Role == provider.RoleUser && msg.Content[0].ToolResult != nil {
			if strings.HasPrefix(msg.Content[0].ToolResult.Content, "[Previous: used bash]") {
				found = true
			}
		}
	}
	if !found {
		t.Error("expected old tool result to be compacted")
	}
}

func TestMicrocompact_ShortResultsPreserved(t *testing.T) {
	a := &Agent{maxHistory: 40}
	// 10 messages with short content (<= 200 chars)
	for i := range 5 {
		a.history = append(a.history,
			provider.NewTextMessage(provider.RoleAssistant, fmt.Sprintf("resp %d", i)),
			toolResultMsg("bash", "ok"),
		)
	}
	a.microcompact()
	for _, msg := range a.history {
		if msg.Role == provider.RoleUser && msg.Content[0].ToolResult != nil {
			if msg.Content[0].ToolResult.Content != "ok" {
				t.Errorf("short result was modified: %q", msg.Content[0].ToolResult.Content)
			}
		}
	}
}

func TestMicrocompact_FileRefPreserved(t *testing.T) {
	a := &Agent{maxHistory: 40}
	content := strings.Repeat("x", 300) + "\n[Full output: /tmp/scratch/test.txt]"
	for i := range 5 {
		a.history = append(a.history,
			provider.NewTextMessage(provider.RoleAssistant, fmt.Sprintf("resp %d", i)),
			toolResultMsg("bash", content),
		)
	}
	a.microcompact()
	for i := 0; i < 4; i++ {
		msg := a.history[i]
		if msg.Role == provider.RoleUser && msg.Content[0].ToolResult != nil {
			c := msg.Content[0].ToolResult.Content
			if strings.HasPrefix(c, "[Previous:") && !strings.Contains(c, "[Full output:") {
				t.Error("file reference lost during compaction")
			}
		}
	}
}

func TestMicrocompact_TextBlocksUntouched(t *testing.T) {
	a := &Agent{maxHistory: 40}
	bigContent := strings.Repeat("x", 300)
	for i := range 5 {
		a.history = append(a.history,
			provider.NewTextMessage(provider.RoleUser, bigContent),
			provider.NewTextMessage(provider.RoleAssistant, fmt.Sprintf("resp %d", i)),
		)
	}
	a.microcompact()
	// Text blocks should never be modified.
	for _, msg := range a.history {
		for _, block := range msg.Content {
			if block.Type == provider.ContentText && strings.HasPrefix(block.Text, "[Previous:") {
				t.Error("text block was incorrectly compacted")
			}
		}
	}
}

func TestMicrocompact_AssistantUntouched(t *testing.T) {
	a := &Agent{maxHistory: 40}
	bigContent := strings.Repeat("x", 300)
	for range 5 {
		a.history = append(a.history,
			provider.NewTextMessage(provider.RoleAssistant, bigContent),
			toolResultMsg("bash", bigContent),
		)
	}
	a.microcompact()
	for _, msg := range a.history {
		if msg.Role == provider.RoleAssistant {
			if strings.HasPrefix(msg.Content[0].Text, "[Previous:") {
				t.Error("assistant message was incorrectly compacted")
			}
		}
	}
}

func TestMicrocompact_EmptyHistory(t *testing.T) {
	a := &Agent{maxHistory: 40}
	a.microcompact() // should not panic
}

func TestMicrocompact_ShortHistory(t *testing.T) {
	a := &Agent{maxHistory: 40}
	a.history = append(a.history,
		provider.NewTextMessage(provider.RoleAssistant, "hi"),
		toolResultMsg("bash", strings.Repeat("x", 300)),
	)
	a.microcompact()
	// Too short for compaction — content should be unchanged.
	if a.history[1].Content[0].ToolResult.Content != strings.Repeat("x", 300) {
		t.Error("short history should not be compacted")
	}
}

func TestMicrocompact_Idempotent(t *testing.T) {
	a := &Agent{maxHistory: 40}
	bigContent := strings.Repeat("x", 300)
	for i := range 5 {
		a.history = append(a.history,
			provider.NewTextMessage(provider.RoleAssistant, fmt.Sprintf("resp %d", i)),
			toolResultMsg("bash", bigContent),
		)
	}
	a.microcompact()
	// Save state after first compaction.
	var firstPass []string
	for _, msg := range a.history {
		if msg.Role == provider.RoleUser && msg.Content[0].ToolResult != nil {
			firstPass = append(firstPass, msg.Content[0].ToolResult.Content)
		}
	}
	// Run again.
	a.microcompact()
	var secondPass []string
	for _, msg := range a.history {
		if msg.Role == provider.RoleUser && msg.Content[0].ToolResult != nil {
			secondPass = append(secondPass, msg.Content[0].ToolResult.Content)
		}
	}
	if len(firstPass) != len(secondPass) {
		t.Fatalf("length changed: %d vs %d", len(firstPass), len(secondPass))
	}
	for idx := range firstPass {
		if firstPass[idx] != secondPass[idx] {
			t.Errorf("index %d changed: %q vs %q", idx, firstPass[idx], secondPass[idx])
		}
	}
}
