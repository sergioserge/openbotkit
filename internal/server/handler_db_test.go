package server

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
)

func TestDBHandler_ValidQuery(t *testing.T) {
	if _, err := exec.LookPath("sqlite3"); err != nil {
		t.Skip("sqlite3 not in PATH")
	}

	dir := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", dir)

	gmailDir := filepath.Join(dir, "gmail")
	if err := os.MkdirAll(gmailDir, 0700); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(gmailDir, "data.db")
	c := exec.Command("sqlite3", dbPath,
		"CREATE TABLE test (name TEXT, age INT); INSERT INTO test VALUES ('alice', 30); INSERT INTO test VALUES ('bob', 25);")
	if err := c.Run(); err != nil {
		t.Fatalf("create test db: %v", err)
	}

	s := &Server{cfg: config.Default()}

	body := `{"sql": "SELECT name, age FROM test ORDER BY name"}`
	req := httptest.NewRequest("POST", "/api/db/gmail", strings.NewReader(body))
	req.SetPathValue("source", "gmail")
	rec := httptest.NewRecorder()

	s.handleDB(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp dbResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Columns) != 2 || resp.Columns[0] != "name" {
		t.Fatalf("unexpected columns: %v", resp.Columns)
	}
	if len(resp.Rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(resp.Rows))
	}
	if resp.Rows[0][0] != "alice" {
		t.Fatalf("expected alice, got %q", resp.Rows[0][0])
	}
}

func TestDBHandler_UnknownSource(t *testing.T) {
	s := &Server{cfg: config.Default()}

	body := `{"sql": "SELECT 1"}`
	req := httptest.NewRequest("POST", "/api/db/unknown", strings.NewReader(body))
	req.SetPathValue("source", "unknown")
	rec := httptest.NewRecorder()

	s.handleDB(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestDBHandler_NonSelectQuery(t *testing.T) {
	s := &Server{cfg: config.Default()}

	body := `{"sql": "DELETE FROM test"}`
	req := httptest.NewRequest("POST", "/api/db/gmail", strings.NewReader(body))
	req.SetPathValue("source", "gmail")
	rec := httptest.NewRecorder()

	s.handleDB(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDBHandler_EmptyResult(t *testing.T) {
	if _, err := exec.LookPath("sqlite3"); err != nil {
		t.Skip("sqlite3 not in PATH")
	}

	dir := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", dir)

	gmailDir := filepath.Join(dir, "gmail")
	if err := os.MkdirAll(gmailDir, 0700); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(gmailDir, "data.db")
	c := exec.Command("sqlite3", dbPath, "CREATE TABLE test (name TEXT);")
	if err := c.Run(); err != nil {
		t.Fatalf("create test db: %v", err)
	}

	s := &Server{cfg: config.Default()}

	body := `{"sql": "SELECT name FROM test"}`
	req := httptest.NewRequest("POST", "/api/db/gmail", strings.NewReader(body))
	req.SetPathValue("source", "gmail")
	rec := httptest.NewRecorder()

	s.handleDB(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp dbResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Rows) != 0 {
		t.Fatalf("expected 0 rows, got %d", len(resp.Rows))
	}
}
