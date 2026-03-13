package tools

import (
	"fmt"
	"strings"
)

// TruncateHead keeps the first maxLines lines (for file_read, web results).
func TruncateHead(s string, maxLines int) string {
	if s == "" || maxLines <= 0 {
		return s
	}
	lines := strings.Split(s, "\n")
	if len(lines) <= maxLines {
		return s
	}
	kept := strings.Join(lines[:maxLines], "\n")
	return kept + fmt.Sprintf("\n...[truncated: showing %d of %d lines]", maxLines, len(lines))
}

// TruncateTail keeps the last maxLines lines (for bash — errors at bottom).
func TruncateTail(s string, maxLines int) string {
	if s == "" || maxLines <= 0 {
		return s
	}
	lines := strings.Split(s, "\n")
	if len(lines) <= maxLines {
		return s
	}
	kept := strings.Join(lines[len(lines)-maxLines:], "\n")
	return fmt.Sprintf("...[truncated: showing %d of %d lines]\n", maxLines, len(lines)) + kept
}

// TruncateHeadTail keeps first headLines + last tailLines lines (for JSON/API responses).
func TruncateHeadTail(s string, headLines, tailLines int) string {
	if s == "" || headLines <= 0 || tailLines <= 0 {
		return s
	}
	lines := strings.Split(s, "\n")
	if len(lines) <= headLines+tailLines {
		return s
	}
	head := strings.Join(lines[:headLines], "\n")
	tail := strings.Join(lines[len(lines)-tailLines:], "\n")
	return head + fmt.Sprintf("\n...[truncated: showing %d+%d of %d lines]\n", headLines, tailLines, len(lines)) + tail
}

// TruncateBytes cuts at the byte limit as a safety fallback.
func TruncateBytes(s string, maxBytes int) string {
	if len(s) <= maxBytes || maxBytes <= 0 {
		return s
	}
	cut := s[:maxBytes]
	// Avoid cutting mid-rune: trim trailing incomplete UTF-8 bytes.
	for len(cut) > 0 && cut[len(cut)-1]&0xC0 == 0x80 {
		cut = cut[:len(cut)-1]
	}
	if len(cut) > 0 && cut[len(cut)-1]&0x80 != 0 {
		cut = cut[:len(cut)-1]
	}
	return cut + fmt.Sprintf("\n...[truncated: showing %dKB of %dKB]", maxBytes/1024, len(s)/1024)
}
