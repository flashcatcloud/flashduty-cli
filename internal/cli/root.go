package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/flashcatcloud/go-flashduty"
	"github.com/spf13/cobra"
	toon "github.com/toon-format/toon-go"
	"golang.org/x/term"

	"github.com/flashcatcloud/flashduty-cli/internal/config"
	"github.com/flashcatcloud/flashduty-cli/internal/output"
	"github.com/flashcatcloud/flashduty-cli/internal/update"
)

// newClientFn creates the go-flashduty client used by all commands.
// Override in tests to inject a stub server.
var newClientFn = defaultNewClient

var (
	flagJSON         bool
	flagNoTrunc      bool
	flagAppKey       string
	flagBaseURL      string
	flagOutputFormat string
)

var updateNotice *update.CheckResult

var rootCmd = &cobra.Command{
	Use:           "flashduty",
	Short:         "Flashduty CLI - incident management from your terminal",
	Long:          "Flashduty CLI - incident management from your terminal.\n\nGet started by running 'flashduty login' to authenticate.",
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
		if _, err := resolveOutputFormat(); err != nil {
			return err
		}
		if cmd.CommandPath() == "flashduty update" {
			return nil
		}
		if !term.IsTerminal(int(os.Stderr.Fd())) {
			return nil
		}
		updateNotice = update.StateHasUpdate(versionStr)
		if update.ShouldCheck(versionStr) {
			go func() {
				_, _ = update.CheckForUpdate(versionStr)
			}()
		}
		return nil
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
	rootCmd.PersistentFlags().BoolVar(&flagJSON, "json", false, "Output as JSON (alias for --output-format json)")
	rootCmd.PersistentFlags().StringVar(&flagOutputFormat, "output-format", "", "Output format: table (default), json, or toon (compact, fewer tokens)")
	rootCmd.PersistentFlags().BoolVar(&flagNoTrunc, "no-trunc", false, "Do not truncate table output")
	rootCmd.PersistentFlags().StringVar(&flagAppKey, "app-key", "", "Override app key")
	rootCmd.PersistentFlags().StringVar(&flagBaseURL, "base-url", "", "Override base URL")
	_ = rootCmd.PersistentFlags().MarkHidden("app-key")
	registerEnumFlag(rootCmd, "output-format", "table", "json", "toon")

	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newLoginCmd())
	rootCmd.AddCommand(newConfigCmd())
	rootCmd.AddCommand(newIncidentCmd())
	rootCmd.AddCommand(newChangeCmd())
	rootCmd.AddCommand(newMemberCmd())
	rootCmd.AddCommand(newTeamCmd())
	rootCmd.AddCommand(newChannelCmd())
	rootCmd.AddCommand(newFieldCmd())
	rootCmd.AddCommand(newTemplateCmd())

	// Phase 1
	rootCmd.AddCommand(newAlertCmd())
	rootCmd.AddCommand(newAlertEventCmd())

	// Phase 2
	rootCmd.AddCommand(newOncallCmd())

	// Phase 3
	rootCmd.AddCommand(newInsightCmd())
	rootCmd.AddCommand(newAuditCmd())

	rootCmd.AddCommand(newWhoamiCmd())
	rootCmd.AddCommand(newUpdateCmd())

	// AI agent sessions (list + transcript export).
	rootCmd.AddCommand(newSessionCmd())

	// Diagnostics entry points (value-add over the raw API).
	rootCmd.AddCommand(newMonitQueryCmd())
	rootCmd.AddCommand(newMonitAgentCmd())

	// Generated commands (full OpenAPI coverage). Registered AFTER curated
	// commands so curated leaves win on any name conflict (see genAddLeaf).
	registerGenerated(rootCmd)

	// session/export is a streaming op excluded from the generated tree; attach
	// its path-is-king leaf to the (now-existing) generated `safari` group so the
	// operation stays reachable at safari session-export.
	attachSafariSessionExport(rootCmd)
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

// newClient creates a go-flashduty client using the current factory.
func newClient() (*flashduty.Client, error) {
	return newClientFn()
}

// defaultNewClient creates a real go-flashduty client from resolved config +
// flag overrides. In broker mode (FLASHDUTY_CRED_FD set — runner-injected), it
// routes all egress over the inherited control fd and sends a sentinel app_key
// the broker overwrites with the real per-person key.
func defaultNewClient() (*flashduty.Client, error) {
	cfg, err := loadResolvedConfig()
	if err != nil {
		return nil, err
	}

	opts := []flashduty.Option{
		flashduty.WithUserAgent("flashduty-cli/" + versionStr),
		flashduty.WithLogger(&silentLogger{}),
	}

	appKey := cfg.AppKey
	if fdStr := os.Getenv("FLASHDUTY_CRED_FD"); fdStr != "" {
		fd, perr := strconv.Atoi(fdStr)
		if perr != nil || fd < 0 {
			return nil, fmt.Errorf("invalid FLASHDUTY_CRED_FD=%q", fdStr)
		}
		hc := newBrokerHTTPClient(fd)
		if hc == nil {
			return nil, errBrokerUnsupported
		}
		opts = append(opts, flashduty.WithHTTPClient(hc))
		appKey = "broker-sentinel" // non-empty: go-flashduty rejects ""; broker overwrites it
	} else if appKey == "" {
		return nil, fmt.Errorf("no app key configured. Run 'flashduty login' or set FLASHDUTY_APP_KEY")
	}

	if cfg.BaseURL != "" && cfg.BaseURL != config.DefaultBaseURL {
		opts = append(opts, flashduty.WithBaseURL(cfg.BaseURL))
	}

	return flashduty.NewClient(appKey, opts...)
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

// resolveOutputFormat maps the global flags to an output.Format. --output-format
// wins when set; otherwise --json selects JSON; otherwise the human table view.
// An unrecognized --output-format value is an error so a typo fails fast rather
// than silently falling back.
func resolveOutputFormat() (output.Format, error) {
	switch strings.ToLower(strings.TrimSpace(flagOutputFormat)) {
	case "table":
		return output.FormatTable, nil
	case "json":
		return output.FormatJSON, nil
	case "toon":
		return output.FormatTOON, nil
	case "":
		if flagJSON {
			return output.FormatJSON, nil
		}
		return output.FormatTable, nil
	default:
		return output.FormatTable, fmt.Errorf("invalid --output-format %q (want table, json, or toon)", flagOutputFormat)
	}
}

// currentOutputFormat returns the resolved format, defaulting to table on the
// error path (the error is surfaced once in PersistentPreRunE, so call sites
// that already passed validation can ignore it).
func currentOutputFormat() output.Format {
	f, _ := resolveOutputFormat()
	return f
}

// marshalStructured serializes v for machine-readable output: indented JSON for
// FormatJSON (byte-compatible with the legacy --json path) and TOON via the
// toon-format encoder for FormatTOON.
func marshalStructured(v any) ([]byte, error) {
	if currentOutputFormat() == output.FormatTOON {
		return toon.Marshal(v)
	}
	return json.MarshalIndent(v, "", "  ")
}

// newPrinter creates a Printer based on global flags.
func newPrinter(w io.Writer) output.Printer {
	if w == nil {
		w = os.Stdout
	}
	return output.NewPrinter(currentOutputFormat(), flagNoTrunc, w)
}

// cmdContext returns the command's context.
func cmdContext(cmd *cobra.Command) context.Context {
	return cmd.Context()
}

// writeResult prints a success message as plain text, or as a structured
// {"message": ...} object in JSON/TOON mode.
func writeResult(w io.Writer, message string) {
	if w == nil {
		w = os.Stdout
	}
	if currentOutputFormat().Structured() {
		out, _ := marshalStructured(map[string]string{"message": message})
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
