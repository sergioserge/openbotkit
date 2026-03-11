package contacts

import "time"

type Contact struct {
	ID          int64
	DisplayName string
	Identities  []Identity
	Aliases     []string
	Interactions []Interaction
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Identity struct {
	ID            int64
	ContactID     int64
	Source        string // "whatsapp", "gmail", "imessage", "applecontacts"
	IdentityType  string // "phone", "email", "wa_jid", "im_handle"
	IdentityValue string // normalized value
	DisplayName   string
	RawValue      string
}

type Interaction struct {
	Channel           string
	MessageCount      int
	LastInteractionAt *time.Time
}

type SearchResult struct {
	Contact      Contact
	MatchScore   int
	MatchedAlias string
}

type SyncResult struct {
	Created int
	Updated int
	Linked  int
	Errors  int
}

type SyncOptions struct {
	Sources []string // empty = all available
}
