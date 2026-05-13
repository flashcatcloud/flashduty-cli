package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	flashduty "github.com/flashcatcloud/flashduty-sdk"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/flashcatcloud/flashduty-cli/internal/config"
)

func newLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Authenticate with Flashduty",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Print("Enter your Flashduty App Key: ")
			raw, err := term.ReadPassword(int(os.Stdin.Fd()))
			fmt.Println()
			if err != nil {
				return fmt.Errorf("failed to read app key: %w", err)
			}

			appKey := string(raw)
			if appKey == "" {
				return fmt.Errorf("app key cannot be empty")
			}

			client, err := flashduty.NewClient(appKey, flashduty.WithLogger(&silentLogger{}))
			if err != nil {
				return fmt.Errorf("invalid app key: %w", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			// Validate by fetching account info
			account, err := client.GetAccountInfo(ctx)
			if err != nil {
				return fmt.Errorf("authentication failed: %w", err)
			}

			// Save to config
			cfg, _ := config.Load()
			cfg.AppKey = appKey
			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			w := cmd.OutOrStdout()
			_, _ = fmt.Fprintf(w, "Logged in successfully.\n")
			_, _ = fmt.Fprintf(w, "  Account:  %s\n", account.AccountName)
			if account.Email != "" {
				_, _ = fmt.Fprintf(w, "  Email:    %s\n", account.Email)
			}
			if account.TimeZone != "" {
				_, _ = fmt.Fprintf(w, "  Timezone: %s\n", account.TimeZone)
			}

			return nil
		},
	}
}
