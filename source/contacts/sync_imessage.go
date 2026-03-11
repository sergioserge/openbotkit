package contacts

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/priyanshujain/openbotkit/store"
)

func syncFromIMessage(contactsDB, imDB *store.DB) (*SyncResult, error) {
	result := &SyncResult{}

	rows, err := imDB.Query("SELECT handle_id, service FROM imessage_handles")
	if err != nil {
		return nil, fmt.Errorf("query imessage handles: %w", err)
	}
	defer rows.Close()

	type imHandle struct {
		handleID string
		service  string
	}
	var handles []imHandle
	for rows.Next() {
		var h imHandle
		if err := rows.Scan(&h.handleID, &h.service); err != nil {
			return nil, fmt.Errorf("scan imessage handle: %w", err)
		}
		handles = append(handles, h)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, h := range handles {
		var identityType, identityValue string
		if strings.Contains(h.handleID, "@") {
			identityType = "email"
			identityValue = NormalizeEmail(h.handleID)
		} else {
			identityType = "phone"
			identityValue = NormalizePhone(h.handleID)
		}
		if identityValue == "" {
			continue
		}

		existing, err := FindContactByIdentity(contactsDB, identityType, identityValue)
		if err != nil {
			slog.Error("contacts: find by imessage handle", "handle", h.handleID, "error", err)
			result.Errors++
			continue
		}

		var contactID int64
		if existing != nil {
			contactID = existing.ID
			result.Linked++
		} else {
			contactID, err = CreateContact(contactsDB, h.handleID)
			if err != nil {
				slog.Error("contacts: create from imessage", "handle", h.handleID, "error", err)
				result.Errors++
				continue
			}
			result.Created++
		}

		if err := UpsertIdentity(contactsDB, &Identity{
			ContactID: contactID, Source: "imessage", IdentityType: identityType,
			IdentityValue: identityValue, RawValue: h.handleID,
		}); err != nil {
			result.Errors++
			continue
		}

		displayName := lookupIMDisplayName(imDB, h.handleID)
		if displayName != "" {
			_ = AddAlias(contactsDB, contactID, displayName, "imessage")
		}

		if err := syncIMessageInteractions(contactsDB, imDB, contactID, h.handleID); err != nil {
			slog.Error("contacts: imessage interactions", "handle", h.handleID, "error", err)
		}
	}

	return result, nil
}

func lookupIMDisplayName(imDB *store.DB, handleID string) string {
	var name sql.NullString
	_ = imDB.QueryRow(
		imDB.Rebind(`SELECT c.display_name FROM imessage_chats c
			WHERE c.is_group = 0 AND c.participants_json LIKE ?
			LIMIT 1`),
		"%"+handleID+"%",
	).Scan(&name)
	if name.Valid && name.String != "" {
		return name.String
	}
	return ""
}

func syncIMessageInteractions(contactsDB, imDB *store.DB, contactID int64, handleID string) error {
	var count int
	var lastAt sql.NullTime
	err := imDB.QueryRow(
		imDB.Rebind("SELECT COUNT(*), MAX(date_utc) FROM imessage_messages WHERE sender_id = ?"),
		handleID,
	).Scan(&count, &lastAt)
	if err != nil {
		return err
	}
	if count == 0 {
		return nil
	}
	var t *time.Time
	if lastAt.Valid {
		t = &lastAt.Time
	}
	return UpsertInteraction(contactsDB, contactID, "imessage", count, t)
}
