package tools

import (
	"context"
	"strings"
	"testing"
)

// mockSandboxRuntime is a test double for SandboxRuntime.
type mockSandboxRuntime struct {
	available bool
	output    string
	err       error
}

func (m *mockSandboxRuntime) Available() bool { return m.available }
func (m *mockSandboxRuntime) Exec(_ context.Context, _, _ string) (string, error) {
	return m.output, m.err
}

func TestDetectRuntime_ReturnsNonNilOnSupportedPlatform(t *testing.T) {
	// DetectRuntime may or may not find a runtime depending on the OS.
	// Just ensure it doesn't panic.
	_ = DetectRuntime()
}

func TestSeatbeltRuntime_Available(t *testing.T) {
	s := &SeatbeltRuntime{}
	// Available() returns a bool — just check it doesn't panic.
	_ = s.Available()
}

func TestBwrapRuntime_Available(t *testing.T) {
	bw := &BwrapRuntime{}
	_ = bw.Available()
}

func TestSeatbeltProfile_ContainsDeny(t *testing.T) {
	profile := seatbeltProfile("/tmp/work", "/tmp/sandbox")
	if !strings.Contains(profile, "(deny default)") {
		t.Error("profile should contain deny default")
	}
	if !strings.Contains(profile, "(deny network*)") {
		t.Error("profile should deny network")
	}
	if !strings.Contains(profile, "(allow file-read*)") {
		t.Error("profile should allow file reads")
	}
}

func TestLanguageInterpreters(t *testing.T) {
	for _, lang := range []string{"python", "node", "bash", "ruby"} {
		if _, ok := LanguageInterpreters[lang]; !ok {
			t.Errorf("missing interpreter for %s", lang)
		}
	}
}

func TestMockSandboxRuntime(t *testing.T) {
	m := &mockSandboxRuntime{available: true, output: "hello\n"}
	if !m.Available() {
		t.Error("mock should be available")
	}
	out, err := m.Exec(context.Background(), "echo hello", "")
	if err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if out != "hello\n" {
		t.Errorf("output = %q, want %q", out, "hello\n")
	}
}

func TestSeatbeltProfile_EmptyWorkDir(t *testing.T) {
	profile := seatbeltProfile("", "/tmp/sandbox")
	if !strings.Contains(profile, "/tmp") {
		t.Error("empty workDir should fall back to /tmp")
	}
}

func TestSeatbeltProfile_DenySSH(t *testing.T) {
	profile := seatbeltProfile("/work", "/tmp/sandbox")
	if !strings.Contains(profile, ".ssh") {
		t.Error("profile should deny .ssh access")
	}
}

func TestSeatbeltProfile_WriteDir(t *testing.T) {
	profile := seatbeltProfile("/work", "/tmp/sandbox-xyz")
	if !strings.Contains(profile, "/tmp/sandbox-xyz") {
		t.Error("profile should allow writes to the specified dir")
	}
	if !strings.Contains(profile, "file-write*") {
		t.Error("profile should have file-write rule")
	}
}
