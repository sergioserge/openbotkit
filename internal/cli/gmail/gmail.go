package gmail

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
	Use:   "gmail",
	Short: "Manage Gmail data source",
}

func init() {
	Cmd.AddCommand(syncCmd)
	Cmd.AddCommand(fetchCmd)
	Cmd.AddCommand(emailsCmd)
	Cmd.AddCommand(attachmentsCmd)
	Cmd.AddCommand(sendCmd)
	Cmd.AddCommand(draftsCmd)
}
