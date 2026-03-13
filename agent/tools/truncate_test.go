package tools

import (
	"strings"
	"testing"
)

func TestTruncateHead_UnderLimit(t *testing.T) {
	input := "line1\nline2\nline3"
	got := TruncateHead(input, 5)
	if got != input {
		t.Errorf("expected passthrough, got %q", got)
	}
}

func TestTruncateHead_ExactLimit(t *testing.T) {
	input := "line1\nline2\nline3"
	got := TruncateHead(input, 3)
	if got != input {
		t.Errorf("expected passthrough at exact limit, got %q", got)
	}
}

func TestTruncateHead_OverLimit(t *testing.T) {
	lines := make([]string, 100)
	for i := range lines {
		lines[i] = "line"
	}
	input := strings.Join(lines, "\n")
	got := TruncateHead(input, 10)

	gotLines := strings.Split(got, "\n")
	// 10 kept lines + 1 marker line = 11
	if len(gotLines) != 11 {
		t.Errorf("got %d lines, want 11", len(gotLines))
	}
	if !strings.Contains(got, "[truncated: showing 10 of 100 lines]") {
		t.Errorf("missing truncation marker in %q", got)
	}
}

func TestTruncateTail_UnderLimit(t *testing.T) {
	input := "line1\nline2"
	got := TruncateTail(input, 5)
	if got != input {
		t.Errorf("expected passthrough, got %q", got)
	}
}

func TestTruncateTail_OverLimit(t *testing.T) {
	lines := make([]string, 100)
	for i := range lines {
		lines[i] = "line"
	}
	input := strings.Join(lines, "\n")
	got := TruncateTail(input, 10)

	if !strings.Contains(got, "[truncated: showing 10 of 100 lines]") {
		t.Errorf("missing truncation marker in %q", got)
	}
	// Marker is first line, then 10 kept lines = 11 total.
	gotLines := strings.Split(got, "\n")
	if len(gotLines) != 11 {
		t.Errorf("got %d lines, want 11", len(gotLines))
	}
}

func TestTruncateHeadTail_UnderLimit(t *testing.T) {
	input := "a\nb\nc\nd\ne"
	got := TruncateHeadTail(input, 3, 3)
	if got != input {
		t.Errorf("expected passthrough (5 <= 3+3), got %q", got)
	}
}

func TestTruncateHeadTail_OverLimit(t *testing.T) {
	lines := make([]string, 100)
	for i := range lines {
		lines[i] = "line"
	}
	input := strings.Join(lines, "\n")
	got := TruncateHeadTail(input, 5, 5)

	if !strings.Contains(got, "[truncated: showing 5+5 of 100 lines]") {
		t.Errorf("missing truncation marker in %q", got)
	}
}

func TestTruncateBytes_UnderLimit(t *testing.T) {
	input := "hello"
	got := TruncateBytes(input, 1024)
	if got != input {
		t.Errorf("expected passthrough, got %q", got)
	}
}

func TestTruncateBytes_OverLimit(t *testing.T) {
	input := strings.Repeat("a", 10000)
	got := TruncateBytes(input, 100)
	if len(got) > 200 { // 100 bytes + marker
		t.Errorf("output too long: %d bytes", len(got))
	}
	if !strings.Contains(got, "[truncated:") {
		t.Errorf("missing truncation marker")
	}
}

func TestTruncate_EmptyString(t *testing.T) {
	if got := TruncateHead("", 10); got != "" {
		t.Errorf("TruncateHead empty = %q", got)
	}
	if got := TruncateTail("", 10); got != "" {
		t.Errorf("TruncateTail empty = %q", got)
	}
	if got := TruncateHeadTail("", 5, 5); got != "" {
		t.Errorf("TruncateHeadTail empty = %q", got)
	}
	if got := TruncateBytes("", 100); got != "" {
		t.Errorf("TruncateBytes empty = %q", got)
	}
}

func TestTruncate_NoNewlines(t *testing.T) {
	// Single huge line — line-based truncation passes through, byte truncation kicks in.
	input := strings.Repeat("x", 100000)
	got := TruncateHead(input, 10)
	if got != input {
		t.Error("single line should pass through TruncateHead")
	}
	got = TruncateBytes(input, 1024)
	if !strings.Contains(got, "[truncated:") {
		t.Error("TruncateBytes should truncate single huge line")
	}
}

func TestTruncateBytes_BinaryContent(t *testing.T) {
	// Non-UTF8 bytes.
	input := string([]byte{0xff, 0xfe, 0x80, 0x81, 0x82, 0x83, 0x84, 0x85})
	got := TruncateBytes(input, 4)
	if !strings.Contains(got, "[truncated:") {
		t.Error("expected truncation marker for binary content")
	}
}
