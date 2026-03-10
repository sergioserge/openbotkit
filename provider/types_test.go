package provider

import (
	"encoding/json"
	"testing"
)

func TestNewTextMessage(t *testing.T) {
	msg := NewTextMessage(RoleUser, "hello")
	if msg.Role != RoleUser {
		t.Errorf("Role = %q", msg.Role)
	}
	if len(msg.Content) != 1 {
		t.Fatalf("Content length = %d", len(msg.Content))
	}
	if msg.Content[0].Type != ContentText {
		t.Errorf("Type = %q", msg.Content[0].Type)
	}
	if msg.Content[0].Text != "hello" {
		t.Errorf("Text = %q", msg.Content[0].Text)
	}
}

func TestTextContent_Empty(t *testing.T) {
	resp := &ChatResponse{}
	if text := resp.TextContent(); text != "" {
		t.Errorf("TextContent = %q, want empty", text)
	}
}

func TestTextContent_MultipleBlocks(t *testing.T) {
	resp := &ChatResponse{
		Content: []ContentBlock{
			{Type: ContentText, Text: "hello "},
			{Type: ContentToolUse, ToolCall: &ToolCall{ID: "1", Name: "bash"}},
			{Type: ContentText, Text: "world"},
		},
	}
	if text := resp.TextContent(); text != "hello world" {
		t.Errorf("TextContent = %q, want %q", text, "hello world")
	}
}

func TestToolCalls_Empty(t *testing.T) {
	resp := &ChatResponse{
		Content: []ContentBlock{{Type: ContentText, Text: "no tools"}},
	}
	if calls := resp.ToolCalls(); len(calls) != 0 {
		t.Errorf("ToolCalls length = %d, want 0", len(calls))
	}
}

func TestEffectiveSystemBlocks_FromBlocks(t *testing.T) {
	req := &ChatRequest{
		SystemBlocks: []SystemBlock{
			{Text: "block1", CacheControl: &CacheControl{Type: "ephemeral"}},
			{Text: "block2"},
		},
		System: "should be ignored",
	}
	blocks := req.EffectiveSystemBlocks()
	if len(blocks) != 2 {
		t.Fatalf("got %d blocks, want 2", len(blocks))
	}
	if blocks[0].Text != "block1" || blocks[0].CacheControl == nil {
		t.Errorf("block 0 = %+v", blocks[0])
	}
	if blocks[1].Text != "block2" || blocks[1].CacheControl != nil {
		t.Errorf("block 1 = %+v", blocks[1])
	}
}

func TestEffectiveSystemBlocks_FromString(t *testing.T) {
	req := &ChatRequest{System: "hello system"}
	blocks := req.EffectiveSystemBlocks()
	if len(blocks) != 1 {
		t.Fatalf("got %d blocks, want 1", len(blocks))
	}
	if blocks[0].Text != "hello system" {
		t.Errorf("text = %q", blocks[0].Text)
	}
	if blocks[0].CacheControl != nil {
		t.Error("expected no cache control for plain System string")
	}
}

func TestEffectiveSystemBlocks_Empty(t *testing.T) {
	req := &ChatRequest{}
	if blocks := req.EffectiveSystemBlocks(); len(blocks) != 0 {
		t.Errorf("got %d blocks, want 0", len(blocks))
	}
}

func TestFullSystemText_Blocks(t *testing.T) {
	req := &ChatRequest{
		SystemBlocks: []SystemBlock{
			{Text: "part1"},
			{Text: "part2"},
		},
	}
	if text := req.FullSystemText(); text != "part1part2" {
		t.Errorf("text = %q, want %q", text, "part1part2")
	}
}

func TestFullSystemText_String(t *testing.T) {
	req := &ChatRequest{System: "hello"}
	if text := req.FullSystemText(); text != "hello" {
		t.Errorf("text = %q", text)
	}
}

func TestFullSystemText_Empty(t *testing.T) {
	req := &ChatRequest{}
	if text := req.FullSystemText(); text != "" {
		t.Errorf("text = %q, want empty", text)
	}
}

func TestToolCalls_Multiple(t *testing.T) {
	resp := &ChatResponse{
		Content: []ContentBlock{
			{Type: ContentToolUse, ToolCall: &ToolCall{ID: "1", Name: "bash", Input: json.RawMessage(`{}`)}},
			{Type: ContentText, Text: "between"},
			{Type: ContentToolUse, ToolCall: &ToolCall{ID: "2", Name: "file_read", Input: json.RawMessage(`{}`)}},
		},
	}
	calls := resp.ToolCalls()
	if len(calls) != 2 {
		t.Fatalf("ToolCalls length = %d, want 2", len(calls))
	}
	if calls[0].Name != "bash" || calls[1].Name != "file_read" {
		t.Errorf("names = %q, %q", calls[0].Name, calls[1].Name)
	}
}
