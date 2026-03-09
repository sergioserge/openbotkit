package daemon

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"time"

	"github.com/priyanshujain/openbotkit/config"
	"github.com/priyanshujain/openbotkit/remote"
	ansrc "github.com/priyanshujain/openbotkit/source/applenotes"
	"github.com/priyanshujain/openbotkit/store"
)

// RunBridge syncs Apple Notes locally and pushes them to the remote server.
// Only works on macOS.
func RunBridge(ctx context.Context, cfg *config.Config, client *remote.Client) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("bridge mode requires macOS (for Apple Notes)")
	}

	if err := config.EnsureSourceDir("applenotes"); err != nil {
		return fmt.Errorf("ensure applenotes dir: %w", err)
	}

	db, err := store.Open(store.Config{
		Driver: cfg.AppleNotes.Storage.Driver,
		DSN:    cfg.AppleNotesDataDSN(),
	})
	if err != nil {
		return fmt.Errorf("open applenotes db: %w", err)
	}
	defer db.Close()

	slog.Info("bridge: starting apple notes sync")

	// Initial sync + push
	bridgeSyncAndPush(db, client)

	ticker := time.NewTicker(appleNotesSyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("bridge: stopping")
			return nil
		case <-ticker.C:
			bridgeSyncAndPush(db, client)
		}
	}
}

func bridgeSyncAndPush(db *store.DB, client *remote.Client) {
	result, err := ansrc.Sync(db, ansrc.SyncOptions{})
	if err != nil {
		slog.Error("bridge: sync error", "error", err)
		return
	}
	slog.Info("bridge: sync complete", "synced", result.Synced, "skipped", result.Skipped, "errors", result.Errors)

	if result.Synced == 0 {
		return
	}

	notes, err := ansrc.ListNotes(db, ansrc.ListOptions{Limit: result.Synced})
	if err != nil {
		slog.Error("bridge: list notes error", "error", err)
		return
	}

	if err := client.AppleNotesPush(notes); err != nil {
		slog.Error("bridge: push error", "error", err)
	} else {
		slog.Info("bridge: pushed notes to remote", "count", len(notes))
	}
}
