package server

import (
	"encoding/json"
	"fmt"
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

func startTestServer(t *testing.T) (*httptest.Server, *config.Config) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", dir)

	cfg := config.Default()
	cfg.Auth = &config.AuthConfig{Username: "test", Password: "test"}

	// Ensure source directories exist
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

	s := &Server{cfg: cfg}
	mux := http.NewServeMux()
	s.routes(mux)

	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	return ts, cfg
}

func TestServerIntegration_HealthNoAuth(t *testing.T) {
	ts, _ := startTestServer(t)

	resp, err := http.Get(ts.URL + "/api/health")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]string
	json.NewDecoder(resp.Body).Decode(&body)
	if body["status"] != "ok" {
		t.Fatalf("expected ok, got %q", body["status"])
	}
}

func TestServerIntegration_AuthRequired(t *testing.T) {
	ts, _ := startTestServer(t)

	// Request without auth
	req, _ := http.NewRequest("GET", ts.URL+"/api/memory", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 without auth, got %d", resp.StatusCode)
	}

	// Request with wrong credentials
	req, _ = http.NewRequest("GET", ts.URL+"/api/memory", nil)
	req.SetBasicAuth("wrong", "creds")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 with wrong creds, got %d", resp.StatusCode)
	}

	// Request with correct credentials
	req, _ = http.NewRequest("GET", ts.URL+"/api/memory", nil)
	req.SetBasicAuth("test", "test")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 with correct creds, got %d", resp.StatusCode)
	}
}

func TestServerIntegration_MemoryCRUD(t *testing.T) {
	ts, _ := startTestServer(t)

	// Add
	body := `{"content": "lives in SF", "category": "identity"}`
	req, _ := http.NewRequest("POST", ts.URL+"/api/memory", strings.NewReader(body))
	req.SetBasicAuth("test", "test")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	var addResp memoryAddResponse
	json.NewDecoder(resp.Body).Decode(&addResp)
	resp.Body.Close()
	if addResp.ID == 0 {
		t.Fatal("expected non-zero ID")
	}

	// List
	req, _ = http.NewRequest("GET", ts.URL+"/api/memory", nil)
	req.SetBasicAuth("test", "test")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var items []memoryItem
	json.NewDecoder(resp.Body).Decode(&items)
	resp.Body.Close()
	if len(items) != 1 || items[0].Content != "lives in SF" {
		t.Fatalf("unexpected list result: %+v", items)
	}

	// Delete
	req, _ = http.NewRequest("DELETE", fmt.Sprintf("%s/api/memory/%d", ts.URL, addResp.ID), nil)
	req.SetBasicAuth("test", "test")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}

	// Verify gone
	req, _ = http.NewRequest("GET", ts.URL+"/api/memory", nil)
	req.SetBasicAuth("test", "test")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	json.NewDecoder(resp.Body).Decode(&items)
	resp.Body.Close()
	if len(items) != 0 {
		t.Fatalf("expected empty after delete, got %d", len(items))
	}
}

func TestServerIntegration_DBRoundTrip(t *testing.T) {
	if _, err := exec.LookPath("sqlite3"); err != nil {
		t.Skip("sqlite3 not in PATH")
	}

	ts, cfg := startTestServer(t)

	// Seed a test DB
	dbPath := cfg.GmailDataDSN()
	c := exec.Command("sqlite3", dbPath,
		"CREATE TABLE gmail_emails (id INTEGER PRIMARY KEY, subject TEXT); INSERT INTO gmail_emails VALUES (1, 'Hello');")
	if err := c.Run(); err != nil {
		t.Fatalf("seed db: %v", err)
	}

	body := `{"sql": "SELECT id, subject FROM gmail_emails"}`
	req, _ := http.NewRequest("POST", ts.URL+"/api/db/gmail", strings.NewReader(body))
	req.SetBasicAuth("test", "test")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var dbResp dbResponse
	json.NewDecoder(resp.Body).Decode(&dbResp)
	if len(dbResp.Rows) != 1 || dbResp.Rows[0][1] != "Hello" {
		t.Fatalf("unexpected db response: %+v", dbResp)
	}
}
