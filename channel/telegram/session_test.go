package telegram

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/priyanshujain/openbotkit/config"
	"github.com/priyanshujain/openbotkit/provider"
	historysrc "github.com/priyanshujain/openbotkit/source/history"
	"github.com/priyanshujain/openbotkit/store"
)

func setupTestEnv(t *testing.T) *config.Config {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", dir)
	for _, src := range []string{"history", "user_memory"} {
		os.MkdirAll(filepath.Join(dir, src), 0700)
	}
	return config.Default()
}

func seedHistory(t *testing.T, cfg *config.Config, sessionID string, msgs []historysrc.Message) {
	t.Helper()
	db, err := store.Open(store.Config{Driver: "sqlite", DSN: cfg.HistoryDataDSN()})
	if err != nil {
		t.Fatalf("open history db: %v", err)
	}
	defer db.Close()
	if err := historysrc.Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	convID, err := historysrc.UpsertConversation(db, sessionID, "telegram")
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	for _, m := range msgs {
		if err := historysrc.SaveMessage(db, convID, m.Role, m.Content); err != nil {
			t.Fatalf("save message: %v", err)
		}
	}
}

func TestRestoreSession_RecentSession(t *testing.T) {
	cfg := setupTestEnv(t)
	seedHistory(t, cfg, "tg-abc", []historysrc.Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi there"},
		{Role: "user", Content: "how are you"},
		{Role: "assistant", Content: "fine"},
	})

	bot := &mockBot{}
	ch := NewChannel(bot, 123)
	sm := &SessionManager{cfg: cfg, channel: ch}

	sm.touchSession()

	if sm.sessionID != "tg-abc" {
		t.Fatalf("expected restored session tg-abc, got %q", sm.sessionID)
	}
	if len(sm.history) != 4 {
		t.Fatalf("expected 4 history messages, got %d", len(sm.history))
	}
	if sm.history[0].Content[0].Text != "hello" {
		t.Errorf("history[0] = %q, want 'hello'", sm.history[0].Content[0].Text)
	}
	if len(sm.messages) != 2 {
		t.Fatalf("expected 2 user messages, got %d", len(sm.messages))
	}
}

func TestRestoreSession_ExpiredSession(t *testing.T) {
	cfg := setupTestEnv(t)

	// Seed a session and manually set it to 1 hour ago
	db, err := store.Open(store.Config{Driver: "sqlite", DSN: cfg.HistoryDataDSN()})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	historysrc.Migrate(db)
	historysrc.UpsertConversation(db, "tg-old", "telegram")
	db.Exec("UPDATE history_conversations SET updated_at = datetime('now', '-1 hour') WHERE session_id = 'tg-old'")
	historysrc.SaveMessage(db, 1, "user", "old msg")
	db.Close()

	bot := &mockBot{}
	ch := NewChannel(bot, 123)
	sm := &SessionManager{cfg: cfg, channel: ch}

	sm.touchSession()

	if sm.sessionID == "tg-old" {
		t.Fatal("should not restore expired session")
	}
	if sm.sessionID == "" {
		t.Fatal("should have generated new session ID")
	}
	if len(sm.history) != 0 {
		t.Fatalf("expected empty history, got %d", len(sm.history))
	}
}

func TestRestoreSession_EmptyDB(t *testing.T) {
	cfg := setupTestEnv(t)

	// Migrate but don't seed
	db, err := store.Open(store.Config{Driver: "sqlite", DSN: cfg.HistoryDataDSN()})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	historysrc.Migrate(db)
	db.Close()

	bot := &mockBot{}
	ch := NewChannel(bot, 123)
	sm := &SessionManager{cfg: cfg, channel: ch}

	sm.touchSession()

	if sm.sessionID == "" {
		t.Fatal("should have generated new session ID")
	}
	if len(sm.history) != 0 {
		t.Fatalf("expected empty history, got %d", len(sm.history))
	}
}

func sentTexts(bot *mockBot) []string {
	bot.mu.Lock()
	defer bot.mu.Unlock()
	var texts []string
	for _, c := range bot.sent {
		if msg, ok := c.(tgbotapi.MessageConfig); ok {
			texts = append(texts, msg.Text)
		}
	}
	return texts
}

func TestStartCommand_ResetsSession(t *testing.T) {
	cfg := setupTestEnv(t)
	bot := &mockBot{}
	ch := NewChannel(bot, 123)
	sm := &SessionManager{cfg: cfg, channel: ch}

	sm.sessionID = "tg-abc"
	sm.history = []provider.Message{
		provider.NewTextMessage(provider.RoleUser, "hello"),
		provider.NewTextMessage(provider.RoleAssistant, "hi"),
	}
	sm.messages = []string{"hello"}

	sm.handleMessage(context.Background(), "/start")

	if sm.sessionID != "" {
		t.Fatalf("expected cleared sessionID, got %q", sm.sessionID)
	}
	if sm.history != nil {
		t.Fatalf("expected nil history, got %d messages", len(sm.history))
	}

	texts := sentTexts(bot)
	found := false
	for _, txt := range texts {
		if strings.Contains(txt, "Session reset") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected reset confirmation message, got: %v", texts)
	}
}

func TestStartCommand_WithSuffix(t *testing.T) {
	cfg := setupTestEnv(t)
	bot := &mockBot{}
	ch := NewChannel(bot, 123)
	sm := &SessionManager{cfg: cfg, channel: ch}
	sm.sessionID = "tg-xyz"

	sm.handleMessage(context.Background(), "/start now")

	if sm.sessionID != "" {
		t.Fatalf("expected cleared sessionID, got %q", sm.sessionID)
	}
	texts := sentTexts(bot)
	found := false
	for _, txt := range texts {
		if strings.Contains(txt, "Session reset") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected reset confirmation, got: %v", texts)
	}
}
