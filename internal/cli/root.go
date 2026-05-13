package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	flashduty "github.com/flashcatcloud/flashduty-sdk"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/flashcatcloud/flashduty-cli/internal/config"
	"github.com/flashcatcloud/flashduty-cli/internal/output"
	"github.com/flashcatcloud/flashduty-cli/internal/update"
)

// flashdutyClient defines the SDK operations used by CLI commands.
type flashdutyClient interface {
	// === Account / Member ===
	GetAccountInfo(ctx context.Context) (*flashduty.AccountInfo, error)
	GetMemberInfo(ctx context.Context) (*flashduty.MemberInfo, error)

	// === EXISTING ===
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

	// === PHASE 1: Incident additions ===
	GetIncidentDetail(ctx context.Context, input *flashduty.GetIncidentDetailInput) (*flashduty.GetIncidentDetailOutput, error)
	GetIncidentFeed(ctx context.Context, input *flashduty.GetIncidentFeedInput) (*flashduty.GetIncidentFeedOutput, error)
	ListPostMortems(ctx context.Context, input *flashduty.ListPostMortemsInput) (*flashduty.ListPostMortemsOutput, error)
	MergeIncidents(ctx context.Context, input *flashduty.MergeIncidentsInput) error
	SnoozeIncidents(ctx context.Context, input *flashduty.SnoozeIncidentsInput) error
	ReopenIncidents(ctx context.Context, incidentIDs []string) error
	ReassignIncidents(ctx context.Context, input *flashduty.ReassignIncidentsInput) error

	// === PHASE 1: Alert additions ===
	ListAlerts(ctx context.Context, input *flashduty.ListAlertsInput) (*flashduty.ListAlertsOutput, error)
	GetAlertDetail(ctx context.Context, input *flashduty.GetAlertDetailInput) (*flashduty.GetAlertDetailOutput, error)
	ListAlertEvents(ctx context.Context, input *flashduty.ListAlertEventsInput) (*flashduty.ListAlertEventsOutput, error)
	MergeAlertsToIncident(ctx context.Context, input *flashduty.MergeAlertsInput) error
	GetAlertFeed(ctx context.Context, input *flashduty.GetAlertFeedInput) (*flashduty.GetAlertFeedOutput, error)
	ListAlertEventsGlobal(ctx context.Context, input *flashduty.ListAlertEventsGlobalInput) (*flashduty.ListAlertEventsGlobalOutput, error)

	// === PHASE 2: OnCall + Change ===
	ListSchedulesWithSlots(ctx context.Context, input *flashduty.ListSchedulesWithSlotsInput) (*flashduty.ListSchedulesWithSlotsOutput, error)
	GetScheduleDetail(ctx context.Context, input *flashduty.GetScheduleDetailInput) (*flashduty.GetScheduleDetailOutput, error)
	QueryChangeTrend(ctx context.Context, input *flashduty.QueryChangeTrendInput) (*flashduty.QueryChangeTrendOutput, error)

	// === PHASE 3: Insight + Admin ===
	QueryInsightByTeam(ctx context.Context, input *flashduty.InsightQueryInput) (*flashduty.QueryInsightByTeamOutput, error)
	QueryInsightByChannel(ctx context.Context, input *flashduty.InsightQueryInput) (*flashduty.QueryInsightByChannelOutput, error)
	QueryInsightByResponder(ctx context.Context, input *flashduty.InsightQueryInput) (*flashduty.QueryInsightByResponderOutput, error)
	QueryInsightAlertTopK(ctx context.Context, input *flashduty.QueryInsightAlertTopKInput) (*flashduty.QueryInsightAlertTopKOutput, error)
	QueryInsightIncidentList(ctx context.Context, input *flashduty.QueryInsightIncidentListInput) (*flashduty.QueryInsightIncidentListOutput, error)
	QueryNotificationTrend(ctx context.Context, input *flashduty.QueryNotificationTrendInput) (*flashduty.QueryNotificationTrendOutput, error)
	SearchAuditLogs(ctx context.Context, input *flashduty.SearchAuditLogsInput) (*flashduty.SearchAuditLogsOutput, error)

	// === PHASE 4: Status Page Migration ===
	StartStatusPageMigration(ctx context.Context, input *flashduty.StartStatusPageMigrationInput) (*flashduty.StartStatusPageMigrationOutput, error)
	StartStatusPageEmailSubscriberMigration(ctx context.Context, input *flashduty.StartStatusPageEmailSubscriberMigrationInput) (*flashduty.StartStatusPageMigrationOutput, error)
	GetStatusPageMigrationStatus(ctx context.Context, jobID string) (*flashduty.StatusPageMigrationJob, error)
	CancelStatusPageMigration(ctx context.Context, jobID string) error
}

// newClientFn creates a flashdutyClient. Override in tests to inject a mock.
var newClientFn = defaultNewClient

var (
	flagJSON    bool
	flagNoTrunc bool
	flagAppKey  string
	flagBaseURL string
)

var updateNotice *update.CheckResult

var rootCmd = &cobra.Command{
	Use:           "flashduty",
	Short:         "Flashduty CLI - incident management from your terminal",
	Long:          "Flashduty CLI - incident management from your terminal.\n\nGet started by running 'flashduty login' to authenticate.",
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRun: func(cmd *cobra.Command, _ []string) {
		if cmd.CommandPath() == "flashduty update" {
			return
		}
		if !term.IsTerminal(int(os.Stderr.Fd())) {
			return
		}
		updateNotice = update.StateHasUpdate(versionStr)
		if update.ShouldCheck(versionStr) {
			go func() {
				_, _ = update.CheckForUpdate(versionStr)
			}()
		}
	},
	PersistentPostRun: func(_ *cobra.Command, _ []string) {
		if updateNotice == nil {
			return
		}
		_, _ = fmt.Fprintf(os.Stderr, "\nA new version of flashduty is available: v%s -> %s\n",
			update.StripV(updateNotice.CurrentVersion), updateNotice.LatestVersion)
		_, _ = fmt.Fprintf(os.Stderr, "To update, run: flashduty update\n")
	},
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

	// Phase 1
	rootCmd.AddCommand(newAlertCmd())
	rootCmd.AddCommand(newAlertEventCmd())
	rootCmd.AddCommand(newPostmortemCmd())

	// Phase 2
	rootCmd.AddCommand(newOncallCmd())

	// Phase 3
	rootCmd.AddCommand(newInsightCmd())
	rootCmd.AddCommand(newAuditCmd())

	rootCmd.AddCommand(newWhoamiCmd())
	rootCmd.AddCommand(newUpdateCmd())
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
