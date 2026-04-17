package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/flashcatcloud/flashduty-cli/internal/config"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage CLI configuration",
	}
	cmd.AddCommand(newConfigShowCmd())
	cmd.AddCommand(newConfigSetCmd())
	return cmd
}

func newConfigShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Print current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			displayKey := config.MaskKey(cfg.AppKey)
			if cfg.AppKey == "" {
				displayKey = "(not set)"
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "app_key:  %s %s\n", displayKey, config.ConfigSource("app_key"))
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "base_url: %s %s\n", cfg.BaseURL, config.ConfigSource("base_url"))
			return nil
		},
	}
}

func newConfigSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Long:  "Supported keys: app_key, base_url",
		Args:  requireArgs("key", "value"),
		RunE: func(cmd *cobra.Command, args []string) error {
			key, value := args[0], args[1]

			cfg, err := config.Load()
			if err != nil {
				return err
			}

			switch key {
			case "app_key":
				cfg.AppKey = value
			case "base_url":
				cfg.BaseURL = value
			default:
				return fmt.Errorf("unknown config key %q (supported: app_key, base_url)", key)
			}

			if err := config.Save(cfg); err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Set %s successfully.\n", key)
			return nil
		},
	}
}
