package tghtml

import (
	"bytes"

	"github.com/leonid-shevtsov/telegold"
	"github.com/yuin/goldmark"
)

// Convert transforms standard Markdown into Telegram-compatible HTML.
// Returns the original string unchanged if conversion fails.
var converter = goldmark.New(goldmark.WithRenderer(telegold.NewRenderer()))

func Convert(md string) string {
	var buf bytes.Buffer
	if err := converter.Convert([]byte(md), &buf); err != nil {
		return md
	}
	return buf.String()
}
