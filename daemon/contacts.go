package daemon

import (
	"context"
	"log/slog"
	"time"

	"github.com/priyanshujain/openbotkit/config"
	contactsrc "github.com/priyanshujain/openbotkit/source/contacts"
	"github.com/priyanshujain/openbotkit/store"
)

const contactsSyncInterval = 5 * time.Minute

func runContactsSync(ctx context.Context, cfg *config.Config) <-chan error {
	errCh := make(chan error, 1)

	go func() {
		defer close(errCh)

		if err := config.EnsureSourceDir("contacts"); err != nil {
			slog.Error("contacts: failed to create dir", "error", err)
			errCh <- err
			return
		}

		db, err := store.Open(store.Config{
			Driver: cfg.Contacts.Storage.Driver,
			DSN:    cfg.ContactsDataDSN(),
		})
		if err != nil {
			slog.Error("contacts: failed to open db", "error", err)
			errCh <- err
			return
		}
		defer db.Close()

		sourceDBs := openSourceDBs(cfg)
		defer closeSourceDBs(sourceDBs)

		if len(sourceDBs) == 0 {
			slog.Info("contacts: no linked sources, skipping sync")
			return
		}

		syncContacts(db, sourceDBs)

		ticker := time.NewTicker(contactsSyncInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				slog.Info("contacts: stopping sync")
				return
			case <-ticker.C:
				syncContacts(db, sourceDBs)
			}
		}
	}()

	return errCh
}

func syncContacts(db *store.DB, sourceDBs map[string]*store.DB) {
	result, err := contactsrc.Sync(db, sourceDBs, contactsrc.SyncOptions{})
	if err != nil {
		slog.Error("contacts: sync error", "error", err)
		return
	}
	slog.Info("contacts: sync complete",
		"created", result.Created, "linked", result.Linked, "errors", result.Errors)
}

func openSourceDBs(cfg *config.Config) map[string]*store.DB {
	dbs := make(map[string]*store.DB)
	sources := []struct {
		name   string
		driver string
		dsn    string
	}{
		{"whatsapp", cfg.WhatsApp.Storage.Driver, cfg.WhatsAppDataDSN()},
		{"gmail", cfg.Gmail.Storage.Driver, cfg.GmailDataDSN()},
		{"imessage", cfg.IMessage.Storage.Driver, cfg.IMessageDataDSN()},
	}
	for _, s := range sources {
		if !config.IsSourceLinked(s.name) {
			continue
		}
		sdb, err := store.Open(store.Config{Driver: s.driver, DSN: s.dsn})
		if err != nil {
			slog.Warn("contacts: could not open source db", "source", s.name, "error", err)
			continue
		}
		dbs[s.name] = sdb
	}
	return dbs
}

func closeSourceDBs(dbs map[string]*store.DB) {
	for _, db := range dbs {
		db.Close()
	}
}
