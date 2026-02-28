package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/priyanshujain/openbotkit/config"
	"github.com/priyanshujain/openbotkit/provider/google"
	"github.com/spf13/cobra"
)

var googleCmd = &cobra.Command{
	Use:   "google",
	Short: "Manage Google account authentication and scopes",
	RunE:  googleInteractiveRun,
}

var googleLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate a Google account via OAuth2",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		if err := config.EnsureProviderDir("google"); err != nil {
			return fmt.Errorf("create provider dir: %w", err)
		}

		scopeStr, _ := cmd.Flags().GetString("scopes")
		emailHint, _ := cmd.Flags().GetString("email")

		scopes := parseScopes(scopeStr)
		if len(scopes) == 0 {
			return fmt.Errorf("--scopes is required (e.g. --scopes gmail.readonly)")
		}

		gp := google.New(google.Config{
			CredentialsFile: cfg.GoogleCredentialsFile(),
			TokenDBPath:     cfg.GoogleTokenDBPath(),
		})

		ctx := context.Background()
		email, err := gp.GrantScopes(ctx, emailHint, expandScopes(scopes))
		if err != nil {
			return fmt.Errorf("login failed: %w", err)
		}

		fmt.Printf("Authenticated as %s\n", email)
		fmt.Printf("Granted: %s\n", strings.Join(scopes, ", "))
		return nil
	},
}

var googleRevokeCmd = &cobra.Command{
	Use:   "revoke",
	Short: "Revoke specific scopes for a Google account",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		emailFlag, _ := cmd.Flags().GetString("email")
		scopeStr, _ := cmd.Flags().GetString("scopes")

		if emailFlag == "" {
			return fmt.Errorf("--email is required")
		}
		scopes := parseScopes(scopeStr)
		if len(scopes) == 0 {
			return fmt.Errorf("--scopes is required")
		}

		gp := google.New(google.Config{
			CredentialsFile: cfg.GoogleCredentialsFile(),
			TokenDBPath:     cfg.GoogleTokenDBPath(),
		})

		ctx := context.Background()
		if err := gp.RevokeScopes(ctx, emailFlag, expandScopes(scopes)); err != nil {
			return fmt.Errorf("revoke failed: %w", err)
		}

		fmt.Printf("Revoked %s for %s\n", strings.Join(scopes, ", "), emailFlag)
		return nil
	},
}

var googleStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show Google account authentication status",
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
			fmt.Println("No authenticated Google accounts.")
			return nil
		}

		jsonOut, _ := cmd.Flags().GetBool("json")
		if jsonOut {
			type accountInfo struct {
				Email  string   `json:"email"`
				Scopes []string `json:"scopes"`
			}
			var infos []accountInfo
			for _, a := range accounts {
				scopes, _ := gp.GrantedScopes(ctx, a)
				infos = append(infos, accountInfo{Email: a, Scopes: scopes})
			}
			return json.NewEncoder(os.Stdout).Encode(infos)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ACCOUNT\tSCOPES")
		for _, a := range accounts {
			scopes, _ := gp.GrantedScopes(ctx, a)
			fmt.Fprintf(w, "%s\t%s\n", a, strings.Join(scopes, ", "))
		}
		return w.Flush()
	},
}

// googleInteractiveRun is a placeholder for the TUI flow (Phase 7).
// For now it shows status.
func googleInteractiveRun(cmd *cobra.Command, args []string) error {
	return googleStatusCmd.RunE(cmd, args)
}

func init() {
	googleLoginCmd.Flags().String("scopes", "", "Comma-separated scopes (e.g. gmail.readonly,calendar.readonly)")
	googleLoginCmd.Flags().String("email", "", "Email hint for account selection")

	googleRevokeCmd.Flags().String("email", "", "Account email to revoke scopes for")
	googleRevokeCmd.Flags().String("scopes", "", "Comma-separated scopes to revoke")

	googleStatusCmd.Flags().Bool("json", false, "Output as JSON")

	googleCmd.AddCommand(googleLoginCmd)
	googleCmd.AddCommand(googleRevokeCmd)
	googleCmd.AddCommand(googleStatusCmd)
}

// scopeAliases maps short names to full Google API scope URLs.
var scopeAliases = map[string]string{
	"gmail.readonly":    "https://www.googleapis.com/auth/gmail.readonly",
	"gmail.modify":      "https://www.googleapis.com/auth/gmail.modify",
	"calendar.readonly": "https://www.googleapis.com/auth/calendar.readonly",
	"calendar":          "https://www.googleapis.com/auth/calendar",
}

func parseScopes(s string) []string {
	if s == "" {
		return nil
	}
	var scopes []string
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			scopes = append(scopes, part)
		}
	}
	return scopes
}

func expandScopes(scopes []string) []string {
	expanded := make([]string, 0, len(scopes))
	for _, s := range scopes {
		if full, ok := scopeAliases[s]; ok {
			expanded = append(expanded, full)
		} else {
			expanded = append(expanded, s)
		}
	}
	return expanded
}
