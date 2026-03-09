package whatsapp

import (
	"fmt"

	"github.com/priyanshujain/openbotkit/config"
	"github.com/priyanshujain/openbotkit/remote"
	"github.com/spf13/cobra"
)

func newRemoteClient(cfg *config.Config) (*remote.Client, error) {
	if cfg.Remote == nil || cfg.Remote.Server == "" {
		return nil, fmt.Errorf("remote server not configured — run 'obk setup' to configure")
	}
	return remote.NewClient(cfg.Remote.Server, cfg.Remote.Username, cfg.Remote.Password), nil
}

var Cmd = &cobra.Command{
	Use:   "whatsapp",
	Short: "Manage WhatsApp data source",
}

func init() {
	Cmd.AddCommand(syncCmd)
	Cmd.AddCommand(messagesCmd)
	Cmd.AddCommand(chatsCmd)
	Cmd.AddCommand(contactsCmd)
}
