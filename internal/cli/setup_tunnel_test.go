package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildCallbackURL(t *testing.T) {
	got := buildCallbackURL("panda-new-kit.ngrok-free.app")
	want := "https://panda-new-kit.ngrok-free.app/auth/google/callback"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestWriteNgrokConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ngrok.yml")

	err := writeNgrokConfig(path, "test-token-123", "my-domain.ngrok-free.app")
	if err != nil {
		t.Fatalf("writeNgrokConfig: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "test-token-123") {
		t.Error("config missing authtoken")
	}
	if !strings.Contains(content, "my-domain.ngrok-free.app") {
		t.Error("config missing domain")
	}
	if !strings.Contains(content, "addr: 8085") {
		t.Error("config missing port")
	}
	if !strings.Contains(content, `version: "3"`) {
		t.Error("config missing version")
	}
}
