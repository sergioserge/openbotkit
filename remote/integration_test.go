package remote

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/priyanshujain/openbotkit/config"
	"github.com/priyanshujain/openbotkit/memory"
	"github.com/priyanshujain/openbotkit/store"
)

// startIntegrationServer creates a real test server with in-memory databases
// that mimics the actual server endpoints for contract testing.
func startIntegrationServer(t *testing.T) (*httptest.Server, *config.Config) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", dir)

	cfg := config.Default()
	cfg.Auth = &config.AuthConfig{Username: "test", Password: "test"}

	// Ensure source directories
	for _, src := range []string{"gmail", "whatsapp", "history", "user_memory", "applenotes"} {
		if err := os.MkdirAll(filepath.Join(dir, src), 0700); err != nil {
			t.Fatalf("mkdir %s: %v", src, err)
		}
	}

	// Create and migrate memory DB
	memDSN := cfg.UserMemoryDataDSN()
	memDB, err := store.Open(store.Config{Driver: "sqlite", DSN: memDSN})
	if err != nil {
		t.Fatalf("open memory db: %v", err)
	}
	if err := memory.Migrate(memDB); err != nil {
		t.Fatalf("migrate memory: %v", err)
	}
	memDB.Close()

	mux := http.NewServeMux()
	registerTestRoutes(t, mux, cfg)

	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	return ts, cfg
}

func registerTestRoutes(t *testing.T, mux *http.ServeMux, cfg *config.Config) {
	t.Helper()

	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	mux.HandleFunc("POST /api/db/{source}", func(w http.ResponseWriter, r *http.Request) {
		source := r.PathValue("source")
		var req struct{ SQL string `json:"sql"` }
		json.NewDecoder(r.Body).Decode(&req)

		dsn, err := cfg.SourceDataDSN(source)
		if err != nil {
			w.WriteHeader(400)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		sqlite3, _ := exec.LookPath("sqlite3")
		cmd := exec.Command(sqlite3, "-header", "-separator", "\t", dsn, req.SQL)
		out, err := cmd.Output()
		if err != nil {
			w.WriteHeader(500)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		columns := strings.Split(lines[0], "\t")
		rows := make([][]string, 0)
		for _, line := range lines[1:] {
			if line != "" {
				rows = append(rows, strings.Split(line, "\t"))
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"columns": columns,
			"rows":    rows,
		})
	})

	mux.HandleFunc("GET /api/memory", func(w http.ResponseWriter, r *http.Request) {
		memDB, _ := store.Open(store.Config{Driver: "sqlite", DSN: cfg.UserMemoryDataDSN()})
		defer memDB.Close()
		memory.Migrate(memDB)

		memories, _ := memory.List(memDB)
		items := make([]map[string]interface{}, len(memories))
		for i, m := range memories {
			items[i] = map[string]interface{}{
				"id":         m.ID,
				"content":    m.Content,
				"category":   string(m.Category),
				"source":     m.Source,
				"created_at": m.CreatedAt.Format("2006-01-02 15:04:05"),
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(items)
	})

	mux.HandleFunc("POST /api/memory", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Content  string `json:"content"`
			Category string `json:"category"`
			Source   string `json:"source"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		memDB, _ := store.Open(store.Config{Driver: "sqlite", DSN: cfg.UserMemoryDataDSN()})
		defer memDB.Close()
		memory.Migrate(memDB)

		id, _ := memory.Add(memDB, req.Content, memory.Category(req.Category), req.Source, "")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]int64{"id": id})
	})

	mux.HandleFunc("DELETE /api/memory/{id}", func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		var id int64
		json.Unmarshal([]byte(idStr), &id)

		memDB, _ := store.Open(store.Config{Driver: "sqlite", DSN: cfg.UserMemoryDataDSN()})
		defer memDB.Close()
		memory.Migrate(memDB)
		memory.Delete(memDB, id)

		w.WriteHeader(http.StatusNoContent)
	})
}

func TestClientServerContract_DB(t *testing.T) {
	if _, err := exec.LookPath("sqlite3"); err != nil {
		t.Skip("sqlite3 not in PATH")
	}

	ts, cfg := startIntegrationServer(t)

	// Seed test data
	dbPath := cfg.GmailDataDSN()
	cmd := exec.Command("sqlite3", dbPath,
		"CREATE TABLE gmail_emails (id INTEGER PRIMARY KEY, subject TEXT); INSERT INTO gmail_emails VALUES (1, 'Test Email');")
	if err := cmd.Run(); err != nil {
		t.Fatalf("seed: %v", err)
	}

	client := NewClient(ts.URL, "", "")
	resp, err := client.DB("gmail", "SELECT id, subject FROM gmail_emails")
	if err != nil {
		t.Fatalf("db: %v", err)
	}

	if len(resp.Columns) != 2 || resp.Columns[1] != "subject" {
		t.Fatalf("unexpected columns: %v", resp.Columns)
	}
	if len(resp.Rows) != 1 || resp.Rows[0][1] != "Test Email" {
		t.Fatalf("unexpected rows: %v", resp.Rows)
	}
}

func TestClientServerContract_Memory(t *testing.T) {
	ts, _ := startIntegrationServer(t)

	client := NewClient(ts.URL, "", "")

	// Add
	id, err := client.MemoryAdd("likes Go", "preference", "manual")
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero ID")
	}

	// List
	items, err := client.MemoryList("")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 1 || items[0].Content != "likes Go" {
		t.Fatalf("unexpected list: %v", items)
	}

	// Delete
	if err := client.MemoryDelete(id); err != nil {
		t.Fatalf("delete: %v", err)
	}

	// Verify gone
	items, err = client.MemoryList("")
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(items))
	}
}
