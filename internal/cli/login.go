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

			// Validate by making a test API call
			client, err := flashduty.NewClient(appKey, flashduty.WithLogger(&silentLogger{}))
			if err != nil {
				return fmt.Errorf("invalid app key: %w", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			result, err := client.ListMembers(ctx, &flashduty.ListMembersInput{})
			if err != nil {
				return fmt.Errorf("authentication failed: %w", err)
			}

			// Save to config
			cfg, _ := config.Load()
			cfg.AppKey = appKey
			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			fmt.Printf("Logged in successfully. Account has %d members.\n", result.Total)
			return nil
		},
	}
}
