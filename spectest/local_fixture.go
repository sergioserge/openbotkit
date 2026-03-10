package spectest

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/priyanshujain/openbotkit/agent"
	"github.com/priyanshujain/openbotkit/agent/tools"
	"github.com/priyanshujain/openbotkit/internal/skills"
	"github.com/priyanshujain/openbotkit/internal/testutil"
	"github.com/priyanshujain/openbotkit/provider"
	"github.com/priyanshujain/openbotkit/provider/gemini"
	embeddedSkills "github.com/priyanshujain/openbotkit/skills"
	gmail "github.com/priyanshujain/openbotkit/source/gmail"
	"github.com/priyanshujain/openbotkit/store"
)

type LocalFixture struct {
	dir      string
	provider provider.Provider
	model    string
}

func NewLocalFixture(t *testing.T) *LocalFixture {
	t.Helper()

	if _, err := exec.LookPath("sqlite3"); err != nil {
		t.Skip("sqlite3 not in PATH")
	}

	p, model := requireProvider(t)
	dir := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", dir)

	createSourceDirs(t, dir)
	createGmailDB(t, dir)
	installSkills(t, dir)
	generateIndex(t)
	buildBinary(t, dir)

	t.Setenv("PATH", filepath.Join(dir, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"))

	return &LocalFixture{
		dir:      dir,
		provider: p,
		model:    model,
	}
}

func requireProvider(t *testing.T) (provider.Provider, string) {
	t.Helper()
	testutil.LoadEnv(t)

	if key := os.Getenv("GEMINI_API_KEY"); key != "" {
		return gemini.New(key), "gemini-2.5-flash"
	}
	t.Skip("no LLM API keys set (GEMINI_API_KEY) — skipping spec tests")
	return nil, ""
}

func createSourceDirs(t *testing.T, dir string) {
	t.Helper()
	for _, src := range []string{"gmail", "whatsapp", "history", "user_memory", "applenotes"} {
		if err := os.MkdirAll(filepath.Join(dir, src), 0700); err != nil {
			t.Fatalf("mkdir %s: %v", src, err)
		}
	}
}

func createGmailDB(t *testing.T, dir string) {
	t.Helper()
	dbPath := filepath.Join(dir, "gmail", "data.db")
	db, err := store.Open(store.SQLiteConfig(dbPath))
	if err != nil {
		t.Fatalf("open gmail db: %v", err)
	}
	defer db.Close()
	if err := gmail.Migrate(db); err != nil {
		t.Fatalf("migrate gmail: %v", err)
	}
}

func installSkills(t *testing.T, dir string) {
	t.Helper()
	skillsDir := filepath.Join(dir, "skills")
	for _, name := range []string{"email-read"} {
		destDir := filepath.Join(skillsDir, name)
		if err := os.MkdirAll(destDir, 0700); err != nil {
			t.Fatalf("mkdir skill %s: %v", name, err)
		}
		entries, err := fs.ReadDir(embeddedSkills.FS, name)
		if err != nil {
			t.Fatalf("read embedded skill %s: %v", name, err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			content, err := fs.ReadFile(embeddedSkills.FS, name+"/"+entry.Name())
			if err != nil {
				t.Fatalf("read embedded file %s/%s: %v", name, entry.Name(), err)
			}
			if err := os.WriteFile(filepath.Join(destDir, entry.Name()), content, 0600); err != nil {
				t.Fatalf("write file %s/%s: %v", name, entry.Name(), err)
			}
		}
	}
}

func generateIndex(t *testing.T) {
	t.Helper()
	if err := skills.GenerateIndex(); err != nil {
		t.Fatalf("generate skill index: %v", err)
	}
}

func buildBinary(t *testing.T, dir string) {
	t.Helper()
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0700); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}

	// Find project root (where go.mod lives).
	root, err := findProjectRoot()
	if err != nil {
		t.Fatalf("find project root: %v", err)
	}

	cmd := exec.Command("go", "build", "-o", filepath.Join(binDir, "obk"), ".")
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("go build: %v", err)
	}
}

func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}

func (f *LocalFixture) Agent(t *testing.T) *agent.Agent {
	t.Helper()

	toolReg := tools.NewRegistry()
	toolReg.Register(tools.NewBashTool(30 * time.Second))
	toolReg.Register(&tools.FileReadTool{})
	toolReg.Register(&tools.LoadSkillsTool{})
	toolReg.Register(&tools.SearchSkillsTool{})

	system := buildSystemPrompt()

	return agent.New(f.provider, f.model, toolReg,
		agent.WithSystem(system),
		agent.WithMaxIterations(15),
	)
}

func buildSystemPrompt() string {
	system := `You are a personal AI assistant powered by OpenBotKit. You help users with email, messaging, notes, and other tasks.

You have core tools available: bash (run commands), file_read, load_skills, search_skills.

To handle domain-specific tasks (email, WhatsApp, notes, etc.), first use search_skills to find relevant skills, then use load_skills to get detailed instructions. Skills teach you how to use bash and sqlite3 for specific domains.
`

	idx, err := skills.LoadIndex()
	if err == nil && len(idx.Skills) > 0 {
		system += "\nAvailable skills:\n"
		for _, s := range idx.Skills {
			system += "- " + s.Name + ": " + s.Description + "\n"
		}
	}

	return system
}

func (f *LocalFixture) GivenEmails(t *testing.T, emails []Email) {
	t.Helper()

	dbPath := filepath.Join(f.dir, "gmail", "data.db")
	db, err := store.Open(store.SQLiteConfig(dbPath))
	if err != nil {
		t.Fatalf("open gmail db: %v", err)
	}
	defer db.Close()

	baseTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	for i, e := range emails {
		if e.MessageID == "" {
			e.MessageID = uuid.New().String()
		}
		if e.Account == "" {
			e.Account = "test@example.com"
		}

		gmailEmail := &gmail.Email{
			MessageID: e.MessageID,
			Account:   e.Account,
			From:      e.From,
			To:        e.To,
			Subject:   e.Subject,
			Body:      e.Body,
			Date:      baseTime.Add(time.Duration(i) * time.Hour),
		}
		if _, err := gmail.SaveEmail(db, gmailEmail); err != nil {
			t.Fatalf("save email %d: %v", i, err)
		}
	}
}
