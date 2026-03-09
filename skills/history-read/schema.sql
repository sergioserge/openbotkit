CREATE TABLE IF NOT EXISTS history_conversations (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  session_id TEXT NOT NULL UNIQUE,
  cwd TEXT,
  started_at DATETIME,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS history_messages (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  conversation_id INTEGER REFERENCES history_conversations(id),
  role TEXT NOT NULL,
  content TEXT,
  timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_history_conv_session ON history_conversations(session_id);
CREATE INDEX IF NOT EXISTS idx_history_msgs_conv ON history_messages(conversation_id);
CREATE INDEX IF NOT EXISTS idx_history_msgs_role ON history_messages(role);
