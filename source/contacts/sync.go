package contacts

import (
	"fmt"
	"log/slog"

	"github.com/priyanshujain/openbotkit/store"
)

func Sync(contactsDB *store.DB, sourceDBs map[string]*store.DB, opts SyncOptions) (*SyncResult, error) {
	if err := Migrate(contactsDB); err != nil {
		return nil, fmt.Errorf("migrate contacts schema: %w", err)
	}

	total := &SyncResult{}

	if shouldSync(opts, "whatsapp") {
		if waDB, ok := sourceDBs["whatsapp"]; ok {
			slog.Info("contacts: syncing from whatsapp")
			r, err := syncFromWhatsApp(contactsDB, waDB)
			if err != nil {
				slog.Error("contacts: whatsapp sync failed", "error", err)
				total.Errors++
			} else {
				merge(total, r)
				_ = SaveSyncState(contactsDB, "whatsapp", "")
			}
		}
	}

	if shouldSync(opts, "gmail") {
		if gmailDB, ok := sourceDBs["gmail"]; ok {
			slog.Info("contacts: syncing from gmail")
			r, err := syncFromGmail(contactsDB, gmailDB)
			if err != nil {
				slog.Error("contacts: gmail sync failed", "error", err)
				total.Errors++
			} else {
				merge(total, r)
				_ = SaveSyncState(contactsDB, "gmail", "")
			}
		}
	}

	if shouldSync(opts, "imessage") {
		if imDB, ok := sourceDBs["imessage"]; ok {
			slog.Info("contacts: syncing from imessage")
			r, err := syncFromIMessage(contactsDB, imDB)
			if err != nil {
				slog.Error("contacts: imessage sync failed", "error", err)
				total.Errors++
			} else {
				merge(total, r)
				_ = SaveSyncState(contactsDB, "imessage", "")
			}
		}
	}

	if shouldSync(opts, "applecontacts") {
		slog.Info("contacts: syncing from apple contacts")
		r, err := syncFromAppleContacts(contactsDB)
		if err != nil {
			slog.Error("contacts: apple contacts sync failed", "error", err)
			total.Errors++
		} else {
			merge(total, r)
			_ = SaveSyncState(contactsDB, "applecontacts", "")
		}
	}

	slog.Info("contacts: sync complete",
		"created", total.Created, "updated", total.Updated,
		"linked", total.Linked, "errors", total.Errors)

	return total, nil
}

func shouldSync(opts SyncOptions, source string) bool {
	if len(opts.Sources) == 0 {
		return true
	}
	for _, s := range opts.Sources {
		if s == source {
			return true
		}
	}
	return false
}

func merge(total, r *SyncResult) {
	total.Created += r.Created
	total.Updated += r.Updated
	total.Linked += r.Linked
	total.Errors += r.Errors
}
