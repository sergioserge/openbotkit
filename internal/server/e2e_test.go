package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings" // used in TestE2E_RemoteClient_DBOutputFormat
	"testing"

	"github.com/priyanshujain/openbotkit/config"
	"github.com/priyanshujain/openbotkit/memory"
	"github.com/priyanshujain/openbotkit/remote"
	ansrc "github.com/priyanshujain/openbotkit/source/applenotes"
	"github.com/priyanshujain/openbotkit/store"
)

// startE2EServer creates a real Server with real handlers backed by real SQLite DBs.
// This is a true end-to-end test — no mocks.
func startE2EServer(t *testing.T) (*httptest.Server, *config.Config) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", dir)

	cfg := config.Default()
	cfg.Auth = &config.AuthConfig{Username: "test", Password: "test"}

	for _, src := range []string{"gmail", "whatsapp", "history", "user_memory", "applenotes"} {
		if err := os.MkdirAll(filepath.Join(dir, src), 0700); err != nil {
			t.Fatalf("mkdir %s: %v", src, err)
		}
	}

	// Create and migrate all databases
	memDSN := cfg.UserMemoryDataDSN()
	memDB, err := store.Open(store.Config{Driver: "sqlite", DSN: memDSN})
	if err != nil {
		t.Fatalf("open memory db: %v", err)
	}
	if err := memory.Migrate(memDB); err != nil {
		t.Fatalf("migrate memory: %v", err)
	}
	memDB.Close()

	anDSN := cfg.AppleNotesDataDSN()
	anDB, err := store.Open(store.Config{Driver: "sqlite", DSN: anDSN})
	if err != nil {
		t.Fatalf("open applenotes db: %v", err)
	}
	if err := ansrc.Migrate(anDB); err != nil {
		t.Fatalf("migrate applenotes: %v", err)
	}
	anDB.Close()

	s := &Server{cfg: cfg}
	mux := http.NewServeMux()
	s.routes(mux)

	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	return ts, cfg
}

// TestE2E_RemoteClient_FullFlow tests the complete round-trip:
// RemoteClient → HTTP → Real Server Handlers → Real SQLite → Response → RemoteClient
func TestE2E_RemoteClient_FullFlow(t *testing.T) {
	ts, _ := startE2EServer(t)

	client := remote.NewClient(ts.URL, "test", "test")

	// 1. Health check
	health, err := client.Health()
	if err != nil {
		t.Fatalf("health: %v", err)
	}
	if health["status"] != "ok" {
		t.Fatalf("expected ok, got %q", health["status"])
	}

	// 2. Memory CRUD
	id, err := client.MemoryAdd("lives in San Francisco", "identity", "manual")
	if err != nil {
		t.Fatalf("memory add: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero memory ID")
	}

	id2, err := client.MemoryAdd("prefers dark mode", "preference", "manual")
	if err != nil {
		t.Fatalf("memory add 2: %v", err)
	}

	// List all
	items, err := client.MemoryList("")
	if err != nil {
		t.Fatalf("memory list: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 memories, got %d", len(items))
	}

	// List by category
	items, err = client.MemoryList("identity")
	if err != nil {
		t.Fatalf("memory list by category: %v", err)
	}
	if len(items) != 1 || items[0].Content != "lives in San Francisco" {
		t.Fatalf("unexpected filtered list: %v", items)
	}

	// Delete one
	if err := client.MemoryDelete(id2); err != nil {
		t.Fatalf("memory delete: %v", err)
	}

	// Verify only one left
	items, err = client.MemoryList("")
	if err != nil {
		t.Fatalf("memory list after delete: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 memory after delete, got %d", len(items))
	}
}

// TestE2E_RemoteClient_DBProxy tests the DB proxy with real SQLite through the real server.
func TestE2E_RemoteClient_DBProxy(t *testing.T) {
	if _, err := exec.LookPath("sqlite3"); err != nil {
		t.Skip("sqlite3 not in PATH")
	}

	ts, cfg := startE2EServer(t)

	// Seed the gmail DB with realistic test data
	dbPath := cfg.GmailDataDSN()
	sql := `
		CREATE TABLE gmail_emails (
			id INTEGER PRIMARY KEY,
			account TEXT,
			message_id TEXT,
			thread_id TEXT,
			from_addr TEXT,
			to_addr TEXT,
			subject TEXT,
			date TEXT,
			body TEXT
		);
		INSERT INTO gmail_emails VALUES (1, 'user@gmail.com', 'msg1', 'thread1', 'alice@example.com', 'user@gmail.com', 'Meeting Tomorrow', '2024-01-15 10:00:00', 'Hi, let us meet at 2pm.');
		INSERT INTO gmail_emails VALUES (2, 'user@gmail.com', 'msg2', 'thread2', 'bob@example.com', 'user@gmail.com', 'Project Update', '2024-01-16 14:00:00', 'Here is the latest update.');
		INSERT INTO gmail_emails VALUES (3, 'work@gmail.com', 'msg3', 'thread3', 'charlie@example.com', 'work@gmail.com', 'Invoice #123', '2024-01-17 09:00:00', 'Please find attached.');
	`
	cmd := exec.Command("sqlite3", dbPath, sql)
	if err := cmd.Run(); err != nil {
		t.Fatalf("seed gmail db: %v", err)
	}

	client := remote.NewClient(ts.URL, "test", "test")

	// Query all emails
	resp, err := client.DB("gmail", "SELECT id, from_addr, subject FROM gmail_emails ORDER BY id")
	if err != nil {
		t.Fatalf("db query: %v", err)
	}
	if len(resp.Columns) != 3 {
		t.Fatalf("expected 3 columns, got %d: %v", len(resp.Columns), resp.Columns)
	}
	if len(resp.Rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(resp.Rows))
	}
	if resp.Rows[0][2] != "Meeting Tomorrow" {
		t.Fatalf("expected 'Meeting Tomorrow', got %q", resp.Rows[0][2])
	}

	// Query with filter
	resp, err = client.DB("gmail", "SELECT subject FROM gmail_emails WHERE account = 'work@gmail.com'")
	if err != nil {
		t.Fatalf("filtered query: %v", err)
	}
	if len(resp.Rows) != 1 || resp.Rows[0][0] != "Invoice #123" {
		t.Fatalf("unexpected filtered result: %v", resp.Rows)
	}

	// Query with count
	resp, err = client.DB("gmail", "SELECT COUNT(*) as total FROM gmail_emails")
	if err != nil {
		t.Fatalf("count query: %v", err)
	}
	if resp.Rows[0][0] != "3" {
		t.Fatalf("expected count 3, got %q", resp.Rows[0][0])
	}

	// Reject non-SELECT queries
	_, err = client.DB("gmail", "DROP TABLE gmail_emails")
	if err == nil {
		t.Fatal("expected error for DROP query")
	}

	// Unknown source
	_, err = client.DB("nonexistent", "SELECT 1")
	if err == nil {
		t.Fatal("expected error for unknown source")
	}
}

// TestE2E_RemoteClient_AppleNotesPush tests pushing notes to the server.
func TestE2E_RemoteClient_AppleNotesPush(t *testing.T) {
	if _, err := exec.LookPath("sqlite3"); err != nil {
		t.Skip("sqlite3 not in PATH")
	}

	ts, _ := startE2EServer(t)
	client := remote.NewClient(ts.URL, "test", "test")

	// Push notes
	notes := []map[string]interface{}{
		{
			"apple_id":           "note-1",
			"title":              "Shopping List",
			"body":               "Milk, bread, eggs",
			"folder":             "Notes",
			"folder_id":          "folder-1",
			"account":            "iCloud",
			"password_protected": false,
			"created_at":         "2024-01-15T10:00:00Z",
			"modified_at":        "2024-01-15T10:00:00Z",
		},
		{
			"apple_id":           "note-2",
			"title":              "Meeting Notes",
			"body":               "Discussed project timeline",
			"folder":             "Work",
			"folder_id":          "folder-2",
			"account":            "iCloud",
			"password_protected": false,
			"created_at":         "2024-01-16T14:00:00Z",
			"modified_at":        "2024-01-16T14:00:00Z",
		},
	}

	if err := client.AppleNotesPush(notes); err != nil {
		t.Fatalf("push: %v", err)
	}

	// Verify notes are in the DB by querying through the DB proxy
	resp, err := client.DB("applenotes", "SELECT title FROM applenotes_notes ORDER BY title")
	if err != nil {
		t.Fatalf("db query: %v", err)
	}
	if len(resp.Rows) != 2 {
		t.Fatalf("expected 2 notes, got %d", len(resp.Rows))
	}
	if resp.Rows[0][0] != "Meeting Notes" {
		t.Fatalf("expected 'Meeting Notes', got %q", resp.Rows[0][0])
	}

	// Push again (upsert) with updated body
	notes[0]["body"] = "Milk, bread, eggs, cheese"
	if err := client.AppleNotesPush(notes); err != nil {
		t.Fatalf("re-push: %v", err)
	}

	// Verify the update
	resp, err = client.DB("applenotes", "SELECT body FROM applenotes_notes WHERE apple_id = 'note-1'")
	if err != nil {
		t.Fatalf("verify update: %v", err)
	}
	if resp.Rows[0][0] != "Milk, bread, eggs, cheese" {
		t.Fatalf("expected updated body, got %q", resp.Rows[0][0])
	}
}

// TestE2E_RemoteClient_AuthRequired tests that all protected endpoints require auth.
func TestE2E_RemoteClient_AuthRequired(t *testing.T) {
	ts, _ := startE2EServer(t)

	// Unauthenticated client
	noAuth := remote.NewClient(ts.URL, "", "")

	// Health should work without auth
	health, err := noAuth.Health()
	if err != nil {
		t.Fatalf("health without auth should work: %v", err)
	}
	if health["status"] != "ok" {
		t.Fatal("health should return ok")
	}

	// DB should fail
	_, err = noAuth.DB("gmail", "SELECT 1")
	if err == nil {
		t.Fatal("expected auth error for DB")
	}

	// Memory should fail
	_, err = noAuth.MemoryList("")
	if err == nil {
		t.Fatal("expected auth error for memory list")
	}

	_, err = noAuth.MemoryAdd("test", "preference", "manual")
	if err == nil {
		t.Fatal("expected auth error for memory add")
	}

	// Wrong credentials
	wrongAuth := remote.NewClient(ts.URL, "wrong", "creds")
	_, err = wrongAuth.MemoryList("")
	if err == nil {
		t.Fatal("expected auth error for wrong credentials")
	}

	// Correct credentials
	authed := remote.NewClient(ts.URL, "test", "test")
	items, err := authed.MemoryList("")
	if err != nil {
		t.Fatalf("authed memory list: %v", err)
	}
	if items == nil {
		t.Fatal("expected non-nil response")
	}
}

// TestE2E_RemoteClient_MultiSourceDB tests querying across different source DBs.
func TestE2E_RemoteClient_MultiSourceDB(t *testing.T) {
	if _, err := exec.LookPath("sqlite3"); err != nil {
		t.Skip("sqlite3 not in PATH")
	}

	ts, cfg := startE2EServer(t)
	client := remote.NewClient(ts.URL, "test", "test")

	// Seed gmail DB
	gmailDB := cfg.GmailDataDSN()
	exec.Command("sqlite3", gmailDB,
		"CREATE TABLE gmail_emails (id INTEGER, subject TEXT); INSERT INTO gmail_emails VALUES (1, 'Test');").Run()

	// Seed whatsapp DB
	waDB := cfg.WhatsAppDataDSN()
	exec.Command("sqlite3", waDB,
		"CREATE TABLE whatsapp_chats (jid TEXT, name TEXT); INSERT INTO whatsapp_chats VALUES ('123@s.whatsapp.net', 'Alice');").Run()

	// Seed history DB
	histDB := cfg.HistoryDataDSN()
	exec.Command("sqlite3", histDB,
		"CREATE TABLE history_conversations (id INTEGER, session_id TEXT); INSERT INTO history_conversations VALUES (1, 'session-1');").Run()

	// Query each source through the same client
	tests := []struct {
		source string
		sql    string
		expect string
	}{
		{"gmail", "SELECT subject FROM gmail_emails", "Test"},
		{"whatsapp", "SELECT name FROM whatsapp_chats", "Alice"},
		{"history", "SELECT session_id FROM history_conversations", "session-1"},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			resp, err := client.DB(tt.source, tt.sql)
			if err != nil {
				t.Fatalf("query %s: %v", tt.source, err)
			}
			if len(resp.Rows) != 1 || resp.Rows[0][0] != tt.expect {
				t.Fatalf("expected %q, got %v", tt.expect, resp.Rows)
			}
		})
	}
}

// TestE2E_RemoteClient_DBOutputFormat verifies the wire format between client and server.
func TestE2E_RemoteClient_DBOutputFormat(t *testing.T) {
	if _, err := exec.LookPath("sqlite3"); err != nil {
		t.Skip("sqlite3 not in PATH")
	}

	ts, cfg := startE2EServer(t)

	dbPath := cfg.GmailDataDSN()
	exec.Command("sqlite3", dbPath,
		"CREATE TABLE t (a TEXT, b INT, c REAL); INSERT INTO t VALUES ('hello', 42, 3.14); INSERT INTO t VALUES ('world', 0, NULL);").Run()

	// Test raw HTTP to verify JSON wire format
	body := `{"sql": "SELECT a, b, c FROM t ORDER BY a"}`
	req, _ := http.NewRequest("POST", ts.URL+"/api/db/gmail", strings.NewReader(body))
	req.SetBasicAuth("test", "test")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	var raw json.RawMessage
	json.NewDecoder(resp.Body).Decode(&raw)

	// Verify it's valid JSON with the expected structure
	var parsed struct {
		Columns []string   `json:"columns"`
		Rows    [][]string `json:"rows"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("parse JSON: %v (raw: %s)", err, raw)
	}

	if len(parsed.Columns) != 3 || parsed.Columns[0] != "a" || parsed.Columns[1] != "b" || parsed.Columns[2] != "c" {
		t.Fatalf("unexpected columns: %v", parsed.Columns)
	}
	if len(parsed.Rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(parsed.Rows))
	}
	if parsed.Rows[0][0] != "hello" || parsed.Rows[0][1] != "42" {
		t.Fatalf("unexpected row 0: %v", parsed.Rows[0])
	}
}
