package envload

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Load reads a .env file and sets environment variables for the test.
// It walks up from the working directory looking for .env.
// Values from .env always override existing env vars (scoped to the test).
// In CI where no .env exists, this is a no-op.
func Load(t *testing.T) {
	t.Helper()
	dir, _ := os.Getwd()
	for {
		envPath := filepath.Join(dir, ".env")
		if f, err := os.Open(envPath); err == nil {
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					val := strings.TrimSpace(parts[1])
					t.Setenv(key, val)
				}
			}
			f.Close()
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return
		}
		dir = parent
	}
}
