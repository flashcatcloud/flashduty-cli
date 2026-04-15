package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	flashduty "github.com/flashcatcloud/flashduty-sdk"
	"github.com/spf13/cobra"

	"github.com/flashcatcloud/flashduty-cli/internal/config"
	"github.com/flashcatcloud/flashduty-cli/internal/output"
)

// flashdutyClient defines the SDK operations used by CLI commands.
type flashdutyClient interface {
	ListIncidents(ctx context.Context, input *flashduty.ListIncidentsInput) (*flashduty.ListIncidentsOutput, error)
	GetIncidentTimelines(ctx context.Context, incidentIDs []string) ([]flashduty.IncidentTimelineOutput, error)
	ListIncidentAlerts(ctx context.Context, incidentIDs []string, limit int) ([]flashduty.IncidentAlertsOutput, error)
	ListSimilarIncidents(ctx context.Context, incidentID string, limit int) (*flashduty.ListIncidentsOutput, error)
	CreateIncident(ctx context.Context, input *flashduty.CreateIncidentInput) (any, error)
	UpdateIncident(ctx context.Context, input *flashduty.UpdateIncidentInput) ([]string, error)
	AckIncidents(ctx context.Context, incidentIDs []string) error
	CloseIncidents(ctx context.Context, incidentIDs []string) error
	ListChannels(ctx context.Context, input *flashduty.ListChannelsInput) (*flashduty.ListChannelsOutput, error)
	ListTeams(ctx context.Context, input *flashduty.ListTeamsInput) (*flashduty.ListTeamsOutput, error)
	ListMembers(ctx context.Context, input *flashduty.ListMembersInput) (*flashduty.ListMembersOutput, error)
	ListEscalationRules(ctx context.Context, channelID int64) (*flashduty.ListEscalationRulesOutput, error)
	ListFields(ctx context.Context, input *flashduty.ListFieldsInput) (*flashduty.ListFieldsOutput, error)
	ListChanges(ctx context.Context, input *flashduty.ListChangesInput) (*flashduty.ListChangesOutput, error)
	GetPresetTemplate(ctx context.Context, input *flashduty.GetPresetTemplateInput) (*flashduty.GetPresetTemplateOutput, error)
	ValidateTemplate(ctx context.Context, input *flashduty.ValidateTemplateInput) (*flashduty.ValidateTemplateOutput, error)
	ListStatusPages(ctx context.Context, pageIDs []int64) ([]flashduty.StatusPage, error)
	ListStatusChanges(ctx context.Context, input *flashduty.ListStatusChangesInput) (*flashduty.ListStatusChangesOutput, error)
	CreateStatusIncident(ctx context.Context, input *flashduty.CreateStatusIncidentInput) (any, error)
	CreateChangeTimeline(ctx context.Context, input *flashduty.CreateChangeTimelineInput) error
}

// newClientFn creates a flashdutyClient. Override in tests to inject a mock.
var newClientFn = defaultNewClient

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

// newClient creates a flashdutyClient using the current factory.
func newClient() (flashdutyClient, error) {
	return newClientFn()
}

// defaultNewClient creates a real Flashduty SDK client from resolved config + flag overrides.
func defaultNewClient() (flashdutyClient, error) {
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

// writeResult prints a message as plain text or JSON depending on the --json flag.
func writeResult(w io.Writer, message string) {
	if w == nil {
		w = os.Stdout
	}
	if flagJSON {
		out, _ := json.MarshalIndent(map[string]string{"message": message}, "", "  ")
		_, _ = fmt.Fprintln(w, string(out))
	} else {
		_, _ = fmt.Fprintln(w, message)
	}
}

// silentLogger suppresses all SDK log output for CLI use.
type silentLogger struct{}

func (s *silentLogger) Debug(msg string, keysAndValues ...any) {}
func (s *silentLogger) Info(msg string, keysAndValues ...any)  {}
func (s *silentLogger) Warn(msg string, keysAndValues ...any)  {}
func (s *silentLogger) Error(msg string, keysAndValues ...any) {}
