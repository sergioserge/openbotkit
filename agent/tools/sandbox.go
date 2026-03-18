package tools

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// SandboxRuntime provides OS-level sandboxed command execution.
type SandboxRuntime interface {
	Available() bool
	Exec(ctx context.Context, command, workDir string) (string, error)
}

// LanguageInterpreters maps language names to interpreter commands.
var LanguageInterpreters = map[string]string{
	"python": "python3",
	"node":   "node",
	"bash":   "bash",
	"ruby":   "ruby",
}

// SeatbeltRuntime uses macOS sandbox-exec for sandboxing.
type SeatbeltRuntime struct{}

func (s *SeatbeltRuntime) Available() bool {
	if runtime.GOOS != "darwin" {
		return false
	}
	_, err := exec.LookPath("sandbox-exec")
	return err == nil
}

func (s *SeatbeltRuntime) Exec(ctx context.Context, command, workDir string) (string, error) {
	writeDir := os.TempDir()
	profile := seatbeltProfile(workDir, writeDir)

	cmd := exec.CommandContext(ctx, "sandbox-exec", "-p", profile, "bash", "-c", command)
	if workDir != "" {
		cmd.Dir = workDir
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := stdout.String()
	if stderr.Len() > 0 {
		result += "\nSTDERR:\n" + stderr.String()
	}
	if err != nil {
		return result, fmt.Errorf("sandbox exec: %w", err)
	}
	return result, nil
}

func seatbeltProfile(workDir, writeDir string) string {
	var b strings.Builder
	b.WriteString("(version 1)\n")
	b.WriteString("(deny default)\n")
	b.WriteString("(allow process-exec)\n")
	b.WriteString("(allow process-fork)\n")
	b.WriteString("(allow sysctl-read)\n")
	b.WriteString("(allow mach-lookup)\n")
	b.WriteString("(allow file-read*)\n")
	// Deny reading sensitive directories.
	b.WriteString(`(deny file-read* (subpath "/Users") (require-not (subpath "`)
	if workDir != "" {
		b.WriteString(workDir)
	} else {
		b.WriteString("/tmp")
	}
	b.WriteString("\")))\n")
	fmt.Fprintf(&b, "(deny file-read* (subpath (string-append (param \"HOME\") \"/.ssh\")))\n")
	// Allow writing only to temp/sandbox dir.
	fmt.Fprintf(&b, "(allow file-write* (subpath %q))\n", writeDir)
	// Deny all network access.
	b.WriteString("(deny network*)\n")
	return b.String()
}

// BwrapRuntime uses bubblewrap (Linux) for sandboxing.
type BwrapRuntime struct{}

func (bw *BwrapRuntime) Available() bool {
	if runtime.GOOS != "linux" {
		return false
	}
	_, err := exec.LookPath("bwrap")
	return err == nil
}

func (bw *BwrapRuntime) Exec(ctx context.Context, command, workDir string) (string, error) {
	writeDir, err := os.MkdirTemp("", "obk-sandbox-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(writeDir)

	args := []string{
		"--ro-bind", "/", "/",
		"--bind", writeDir, "/tmp",
		"--dev", "/dev",
		"--proc", "/proc",
		"--unshare-net",
		"--unshare-pid",
		"--die-with-parent",
		"--", "bash", "-c", command,
	}

	cmd := exec.CommandContext(ctx, "bwrap", args...)
	if workDir != "" {
		cmd.Dir = workDir
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		result := stdout.String()
		if stderr.Len() > 0 {
			result += "\nSTDERR:\n" + stderr.String()
		}
		return result, fmt.Errorf("sandbox exec: %w", err)
	}

	result := stdout.String()
	if stderr.Len() > 0 {
		result += "\nSTDERR:\n" + stderr.String()
	}
	return result, nil
}

// DetectRuntime returns the first available sandbox runtime, or nil.
func DetectRuntime() SandboxRuntime {
	runtimes := []SandboxRuntime{
		&SeatbeltRuntime{},
		&BwrapRuntime{},
	}
	for _, r := range runtimes {
		if r.Available() {
			return r
		}
	}
	return nil
}
