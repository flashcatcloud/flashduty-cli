package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/flashcatcloud/flashduty-cli/internal/config"
	"github.com/flashcatcloud/flashduty-cli/internal/output"
	flashduty "github.com/flashcatcloud/flashduty-sdk"
	"github.com/spf13/cobra"
)

var (
	flagJSON    bool
	flagNoTrunc bool
	flagAppKey  string
	flagBaseURL string
)

var rootCmd = &cobra.Command{
	Use:           "flashduty",
	Short:         "Flashduty CLI - incident management from your terminal",
	Long:          "Flashduty CLI - incident management from your terminal.\n\nGet started by running 'flashduty login' to authenticate.",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&flagJSON, "json", false, "Output as JSON")
	rootCmd.PersistentFlags().BoolVar(&flagNoTrunc, "no-trunc", false, "Do not truncate table output")
	rootCmd.PersistentFlags().StringVar(&flagAppKey, "app-key", "", "Override app key")
	rootCmd.PersistentFlags().StringVar(&flagBaseURL, "base-url", "", "Override base URL")
	_ = rootCmd.PersistentFlags().MarkHidden("app-key")

	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newLoginCmd())
	rootCmd.AddCommand(newConfigCmd())
	rootCmd.AddCommand(newIncidentCmd())
	rootCmd.AddCommand(newChangeCmd())
	rootCmd.AddCommand(newMemberCmd())
	rootCmd.AddCommand(newTeamCmd())
	rootCmd.AddCommand(newChannelCmd())
	rootCmd.AddCommand(newEscalationRuleCmd())
	rootCmd.AddCommand(newFieldCmd())
	rootCmd.AddCommand(newStatusPageCmd())
	rootCmd.AddCommand(newTemplateCmd())
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

// newClient creates a Flashduty SDK client from resolved config + flag overrides.
func newClient() (*flashduty.Client, error) {
	cfg, err := loadResolvedConfig()
	if err != nil {
		return nil, err
	}

	if cfg.AppKey == "" {
		return nil, fmt.Errorf("no app key configured. Run 'flashduty login' or set FLASHDUTY_APP_KEY")
	}

	opts := []flashduty.Option{
		flashduty.WithUserAgent("flashduty-cli/" + versionStr),
		flashduty.WithLogger(&silentLogger{}),
	}
	if cfg.BaseURL != "" && cfg.BaseURL != config.DefaultBaseURL {
		opts = append(opts, flashduty.WithBaseURL(cfg.BaseURL))
	}

	return flashduty.NewClient(cfg.AppKey, opts...)
}

func loadResolvedConfig() (*config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	if flagAppKey != "" {
		cfg.AppKey = flagAppKey
	}
	if flagBaseURL != "" {
		cfg.BaseURL = flagBaseURL
	}

	return cfg, nil
}

// newPrinter creates a Printer based on global flags.
func newPrinter(w io.Writer) output.Printer {
	if w == nil {
		w = os.Stdout
	}
	return output.NewPrinter(flagJSON, flagNoTrunc, w)
}

// cmdContext returns the command's context.
func cmdContext(cmd *cobra.Command) context.Context {
	return cmd.Context()
}

// silentLogger suppresses all SDK log output for CLI use.
type silentLogger struct{}

func (s *silentLogger) Debug(msg string, keysAndValues ...any) {}
func (s *silentLogger) Info(msg string, keysAndValues ...any)  {}
func (s *silentLogger) Warn(msg string, keysAndValues ...any)  {}
func (s *silentLogger) Error(msg string, keysAndValues ...any) {}
