package websearch

import (
	"context"
	"testing"
)

func TestWebSearchName(t *testing.T) {
	ws := New(Config{})
	if ws.Name() != "websearch" {
		t.Fatalf("expected 'websearch', got %q", ws.Name())
	}
}

func TestWebSearchStatusNoDB(t *testing.T) {
	ws := New(Config{})
	st, err := ws.Status(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !st.Connected {
		t.Error("expected Connected=true")
	}
	if st.ItemCount != 0 {
		t.Errorf("expected ItemCount=0, got %d", st.ItemCount)
	}
}
