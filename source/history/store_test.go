package history

import (
	"testing"
	"time"
)

func TestUpsertConversation(t *testing.T) {
	db := testDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	id1, err := UpsertConversation(db, "session-001", "/tmp/project")
	if err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	if id1 == 0 {
		t.Fatal("expected non-zero id")
	}

	// Upsert same session should return same id.
	id2, err := UpsertConversation(db, "session-001", "/tmp/project")
	if err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	if id1 != id2 {
		t.Fatalf("expected same id %d, got %d", id1, id2)
	}

	// Different session should get different id.
	id3, err := UpsertConversation(db, "session-002", "/tmp/other")
	if err != nil {
		t.Fatalf("third upsert: %v", err)
	}
	if id3 == id1 {
		t.Fatal("expected different id for different session")
	}
}

func TestSaveMessage(t *testing.T) {
	db := testDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	convID, err := UpsertConversation(db, "session-001", "/tmp")
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}

	if err := SaveMessage(db, convID, "user", "hello"); err != nil {
		t.Fatalf("save user message: %v", err)
	}
	if err := SaveMessage(db, convID, "assistant", "hi there"); err != nil {
		t.Fatalf("save assistant message: %v", err)
	}

	count, err := MessageCountForSession(db, "session-001")
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 messages, got %d", count)
	}
}

func TestCountConversations(t *testing.T) {
	db := testDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	count, err := CountConversations(db)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0, got %d", count)
	}

	UpsertConversation(db, "s1", "/tmp")
	UpsertConversation(db, "s2", "/tmp")

	count, err = CountConversations(db)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2, got %d", count)
	}
}

func TestLastCaptureTime(t *testing.T) {
	db := testDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	ts, err := LastCaptureTime(db)
	if err != nil {
		t.Fatalf("last capture: %v", err)
	}
	if ts != nil {
		t.Fatalf("expected nil, got %v", ts)
	}

	UpsertConversation(db, "s1", "/tmp")

	ts, err = LastCaptureTime(db)
	if err != nil {
		t.Fatalf("last capture: %v", err)
	}
	if ts == nil {
		t.Fatal("expected non-nil timestamp after upsert")
	}
}

func TestMessageCountForSession(t *testing.T) {
	db := testDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	count, err := MessageCountForSession(db, "nonexistent")
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0, got %d", count)
	}
}

func TestLoadRecentSession_EmptyDB(t *testing.T) {
	db := testDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	s, err := LoadRecentSession(db, "telegram", time.Hour)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if s != nil {
		t.Fatalf("expected nil, got %+v", s)
	}
}

func TestLoadRecentSession_RecentSession(t *testing.T) {
	db := testDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	UpsertConversation(db, "tg-abc", "telegram")

	s, err := LoadRecentSession(db, "telegram", time.Hour)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if s == nil {
		t.Fatal("expected session, got nil")
	}
	if s.SessionID != "tg-abc" {
		t.Fatalf("expected tg-abc, got %q", s.SessionID)
	}
}

func TestLoadRecentSession_ExpiredSession(t *testing.T) {
	db := testDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	UpsertConversation(db, "tg-old", "telegram")

	s, err := LoadRecentSession(db, "telegram", 0)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if s != nil {
		t.Fatalf("expected nil for expired session, got %+v", s)
	}
}

func TestLoadRecentSession_MultipleConvos(t *testing.T) {
	db := testDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	UpsertConversation(db, "tg-first", "telegram")
	UpsertConversation(db, "tg-second", "telegram")
	// Make tg-second strictly newer
	db.Exec("UPDATE history_conversations SET updated_at = datetime('now', '+1 second') WHERE session_id = 'tg-second'")

	s, err := LoadRecentSession(db, "telegram", time.Hour)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if s == nil {
		t.Fatal("expected session")
	}
	if s.SessionID != "tg-second" {
		t.Fatalf("expected most recent tg-second, got %q", s.SessionID)
	}
}
