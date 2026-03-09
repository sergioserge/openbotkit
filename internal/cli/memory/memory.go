package memory

import (
	"fmt"

	"github.com/priyanshujain/openbotkit/config"
	"github.com/priyanshujain/openbotkit/remote"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "memory",
	Short: "Manage personal memory",
}

func newRemoteClient(cfg *config.Config) (*remote.Client, error) {
	if cfg.Remote == nil || cfg.Remote.Server == "" {
		return nil, fmt.Errorf("remote server not configured — run 'obk setup' to configure")
	}
	return remote.NewClient(cfg.Remote.Server, cfg.Remote.Username, cfg.Remote.Password), nil
}

func init() {
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(addCmd)
	Cmd.AddCommand(deleteCmd)
}
