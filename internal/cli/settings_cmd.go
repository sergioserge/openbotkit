package cli

import (
	"fmt"

	"github.com/73ai/openbotkit/config"
	settingstui "github.com/73ai/openbotkit/internal/settings/tui"
	"github.com/73ai/openbotkit/provider"
	"github.com/73ai/openbotkit/settings"
	"github.com/spf13/cobra"
)

var settingsCmd = &cobra.Command{
	Use:   "settings",
	Short: "Browse and edit settings",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		svc := settings.New(cfg,
			settings.WithStoreCred(provider.StoreCredential),
			settings.WithLoadCred(provider.LoadCredential),
		)
		return settingstui.Run(svc)
	},
}

func init() {
	rootCmd.AddCommand(settingsCmd)
}
