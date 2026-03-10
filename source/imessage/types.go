package imessage

import "time"

type Config struct{}

type Message struct {
	GUID                  string
	AppleROWID            int64
	Text                  string
	ChatGUID              string
	SenderID              string
	SenderService         string
	IsFromMe              bool
	IsRead                bool
	Date                  time.Time
	DateRead              time.Time
	ReplyToGUID           string
	AssociatedMessageGUID string
	AssociatedMessageType int
	AttachmentsJSON       string
	ChatDisplayName       string
}

type Chat struct {
	GUID            string
	DisplayName     string
	ServiceName     string
	Participants    []string
	IsGroup         bool
	LastMessageDate time.Time
}

type Handle struct {
	ID      string
	Service string
}

type AttachmentMeta struct {
	Filename   string `json:"filename"`
	MIMEType   string `json:"mime_type"`
	TotalBytes int64  `json:"total_bytes"`
}

type SyncOptions struct {
	Full bool
}

type SyncResult struct {
	Synced  int
	Skipped int
	Errors  int
}

type ListOptions struct {
	ChatGUID string
	Limit    int
	Offset   int
}
