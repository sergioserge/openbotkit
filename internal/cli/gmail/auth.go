package gmail

import (
	"context"
	"fmt"

	"github.com/priyanshujain/openbotkit/config"
	"github.com/priyanshujain/openbotkit/provider/google"
	"github.com/spf13/cobra"
	gapi "google.golang.org/api/gmail/v1"
)

var authCmd = &cobra.Command{
	Use:        "auth",
	Short:      "Manage Gmail authentication",
	Deprecated: "use 'obk auth google' instead",
}

var authLoginCmd = &cobra.Command{
	Use:        "login",
	Short:      "Authenticate a Gmail account via OAuth2",
	Deprecated: "use 'obk auth google login' instead",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		if err := config.EnsureProviderDir("google"); err != nil {
			return fmt.Errorf("create provider dir: %w", err)
		}

		gp := google.New(google.Config{
			CredentialsFile: cfg.GoogleCredentialsFile(),
			TokenDBPath:     cfg.GoogleTokenDBPath(),
		})

		ctx := context.Background()
		email, err := gp.GrantScopes(ctx, "", []string{gapi.GmailReadonlyScope})
		if err != nil {
			return fmt.Errorf("login failed: %w", err)
		}

		fmt.Printf("\nSuccessfully authenticated %s\n", email)
		return nil
	},
}

var authLogoutCmd = &cobra.Command{
	Use:        "logout",
	Short:      "Remove stored tokens for a Gmail account",
	Deprecated: "use 'obk auth google revoke' instead",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		account, _ := cmd.Flags().GetString("account")

		gp := google.New(google.Config{
			CredentialsFile: cfg.GoogleCredentialsFile(),
			TokenDBPath:     cfg.GoogleTokenDBPath(),
		})

		ctx := context.Background()

		if account == "" {
			accounts, err := gp.Accounts(ctx)
			if err != nil || len(accounts) == 0 {
				fmt.Println("No authenticated accounts.")
				return nil
			}
			fmt.Println("Authenticated accounts:")
			for i, a := range accounts {
				fmt.Printf("  %d. %s\n", i+1, a)
			}
			return fmt.Errorf("specify --account to logout")
		}

		scopes, err := gp.GrantedScopes(ctx, account)
		if err != nil {
			return fmt.Errorf("get scopes: %w", err)
		}

		if err := gp.RevokeScopes(ctx, account, scopes); err != nil {
			return fmt.Errorf("logout failed: %w", err)
		}
		fmt.Printf("Logged out of %s\n", account)
		return nil
	},
}

var authStatusCmd = &cobra.Command{
	Use:        "status",
	Short:      "Show connected accounts and token state",
	Deprecated: "use 'obk auth google status' instead",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		gp := google.New(google.Config{
			CredentialsFile: cfg.GoogleCredentialsFile(),
			TokenDBPath:     cfg.GoogleTokenDBPath(),
		})

		ctx := context.Background()
		accounts, err := gp.Accounts(ctx)
		if err != nil || len(accounts) == 0 {
			fmt.Println("No authenticated accounts.")
			return nil
		}

		fmt.Println("Authenticated Gmail accounts:")
		for _, a := range accounts {
			scopes, _ := gp.GrantedScopes(ctx, a)
			fmt.Printf("  %s (scopes: %v)\n", a, scopes)
		}
		return nil
	},
}

func init() {
	authLogoutCmd.Flags().String("account", "", "Account to logout")

	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authLogoutCmd)
	authCmd.AddCommand(authStatusCmd)
}
