package tghtml

import (
	"strings"
	"testing"
)

func TestConvert_Bold(t *testing.T) {
	got := Convert("**hello**")
	if !strings.Contains(got, "<b>hello</b>") {
		t.Errorf("expected <b>hello</b>, got %q", got)
	}
}

func TestConvert_Italic(t *testing.T) {
	got := Convert("*hello*")
	if !strings.Contains(got, "<i>hello</i>") {
		t.Errorf("expected <i>hello</i>, got %q", got)
	}
}

func TestConvert_Code(t *testing.T) {
	got := Convert("`code`")
	if !strings.Contains(got, "<code>code</code>") {
		t.Errorf("expected <code>code</code>, got %q", got)
	}
}

func TestConvert_CodeBlock(t *testing.T) {
	got := Convert("```go\nfmt.Println()\n```")
	if !strings.Contains(got, "<pre>") {
		t.Errorf("expected <pre> block, got %q", got)
	}
}

func TestConvert_Link(t *testing.T) {
	got := Convert("[click](https://example.com)")
	if !strings.Contains(got, `<a href="https://example.com">click</a>`) {
		t.Errorf("expected link, got %q", got)
	}
}

func TestConvert_HTMLEscaped(t *testing.T) {
	got := Convert("a < b & c > d")
	if !strings.Contains(got, "&lt;") || !strings.Contains(got, "&amp;") || !strings.Contains(got, "&gt;") {
		t.Errorf("expected HTML escaping, got %q", got)
	}
}

func TestConvert_PlainText(t *testing.T) {
	got := Convert("just plain text")
	if !strings.Contains(got, "just plain text") {
		t.Errorf("expected plain text preserved, got %q", got)
	}
}
