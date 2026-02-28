package memory

import "github.com/priyanshujain/openbotkit/store"

const schemaSQLite = `
CREATE TABLE IF NOT EXISTS memory_conversations (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	session_id TEXT NOT NULL UNIQUE,
	cwd TEXT,
	started_at DATETIME,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS memory_messages (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	conversation_id INTEGER REFERENCES memory_conversations(id),
	role TEXT NOT NULL,
	content TEXT,
	timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_memory_conv_session ON memory_conversations(session_id);
CREATE INDEX IF NOT EXISTS idx_memory_msgs_conv ON memory_messages(conversation_id);
CREATE INDEX IF NOT EXISTS idx_memory_msgs_role ON memory_messages(role);
`

const schemaPostgres = `
CREATE TABLE IF NOT EXISTS memory_conversations (
	id BIGSERIAL PRIMARY KEY,
	session_id TEXT NOT NULL UNIQUE,
	cwd TEXT,
	started_at TIMESTAMPTZ,
	updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS memory_messages (
	id BIGSERIAL PRIMARY KEY,
	conversation_id BIGINT REFERENCES memory_conversations(id),
	role TEXT NOT NULL,
	content TEXT,
	timestamp TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_memory_conv_session ON memory_conversations(session_id);
CREATE INDEX IF NOT EXISTS idx_memory_msgs_conv ON memory_messages(conversation_id);
CREATE INDEX IF NOT EXISTS idx_memory_msgs_role ON memory_messages(role);
`

func Migrate(db *store.DB) error {
	schema := schemaSQLite
	if db.IsPostgres() {
		schema = schemaPostgres
	}
	_, err := db.Exec(schema)
	return err
}
