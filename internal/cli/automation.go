package cli

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/flashcatcloud/go-flashduty"
	"github.com/spf13/cobra"

	"github.com/flashcatcloud/flashduty-cli/internal/timeutil"
)

const automationHTTPPostOnlyCron = "0 0 * * *"
const automationUTCNote = "Convert local wall-clock requests to UTC before passing --at or --cron-expr."

func newAutomationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "automation",
		Short: "Manage AI SRE Automations",
		Long:  "Create, list, update, delete, inspect, and trigger AI SRE Automations.",
		Example: `  flashduty automation create --name "Daily SRE brief" --schedule daily --at 09:30 --prompt "Summarize yesterday's incidents"
  flashduty automation create --name "Webhook triage" --http-post-trigger --prompt-file ./prompt.md
  flashduty automation list --scope all --limit 20
  flashduty automation fire auttrig_123 --token "$TOKEN" --text "manual test"`,
	}

	cmd.AddCommand(newAutomationCreateCmd())
	cmd.AddCommand(newAutomationListCmd())
	cmd.AddCommand(newAutomationGetCmd())
	cmd.AddCommand(newAutomationUpdateCmd())
	cmd.AddCommand(newAutomationDeleteCmd())
	cmd.AddCommand(newAutomationRunsCmd())
	cmd.AddCommand(newAutomationTemplatesCmd())
	cmd.AddCommand(newAutomationFireCmd())
	return cmd
}

func newAutomationCreateCmd() *cobra.Command {
	var (
		name            string
		teamID          int64
		schedule        string
		at              string
		weekday         string
		cronExpr        string
		disabled        bool
		scheduleEnabled = true
		httpPostTrigger bool
		prompt          string
		promptFile      string
		environmentKind string
		environmentID   string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an Automation",
		Long: curatedLong(`Create an AI SRE Automation.

By default the rule is enabled. Use --disabled only when the user explicitly
asks to create it disabled. team_id=0 means personal scope; --team-id >0 creates
the rule under that team. The scope is immutable after creation.

	Schedule helpers build a 5-field UTC cron expression. --at and --cron-expr are
	interpreted in UTC, not the caller's local timezone. Convert local wall-clock
	requests to UTC before passing --at or --cron-expr.

	For HTTP POST-only rules, pass --http-post-trigger without a schedule; the CLI
	sends a valid placeholder cron and disables the schedule trigger.`, "Automations", "RuleWriteCreate"),
		Example: `  flashduty automation create --name "Daily SRE brief" --schedule daily --at 01:30 --prompt "Summarize yesterday's incidents"
  flashduty automation create --name "Weekly noise review" --team-id 123 --schedule weekly --weekday mon --at 02:00 --prompt-file ./prompt.md
  flashduty automation create --name "Webhook triage" --http-post-trigger --prompt "Handle the posted payload"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				taskPrompt, err := resolveAutomationPrompt(cmd, prompt, promptFile)
				if err != nil {
					return err
				}
				name = strings.TrimSpace(name)
				if name == "" {
					return fmt.Errorf("--name is required")
				}
				if strings.TrimSpace(taskPrompt) == "" {
					return fmt.Errorf("one of --prompt or --prompt-file is required")
				}

				effectiveScheduleEnabled := scheduleEnabled
				cron, err := resolveAutomationCreateCron(cmd, schedule, at, weekday, cronExpr, httpPostTrigger, &effectiveScheduleEnabled)
				if err != nil {
					return err
				}

				req := &flashduty.AutomationRuleCreateRequest{
					Name:                   name,
					TeamID:                 teamID,
					CronExpr:               cron,
					Enabled:                !disabled,
					ScheduleTriggerEnabled: flashduty.Bool(effectiveScheduleEnabled),
					HTTPPostTriggerEnabled: httpPostTrigger,
					Prompt:                 taskPrompt,
					EnvironmentKind:        strings.TrimSpace(environmentKind),
					EnvironmentID:          strings.TrimSpace(environmentID),
				}
				out, _, err := ctx.Client.Automations.RuleWriteCreate(cmdContext(ctx.Cmd), req)
				if err != nil {
					return err
				}
				return printGenericResult(ctx, out)
			})
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Automation name")
	cmd.Flags().Int64Var(&teamID, "team-id", 0, "Scope team ID; 0 means personal scope")
	cmd.Flags().StringVar(&schedule, "schedule", "", "UTC schedule helper: hourly, daily, weekly, or cron")
	cmd.Flags().StringVar(&at, "at", "", "UTC time in HH:MM; for hourly schedules, only the minute is used. "+automationUTCNote)
	cmd.Flags().StringVar(&weekday, "weekday", "", "Weekday for weekly schedules: sun, mon, tue, wed, thu, fri, sat, or 0-7")
	cmd.Flags().StringVar(&cronExpr, "cron-expr", "", "Exact 5-field UTC cron expression; overrides --schedule helpers. "+automationUTCNote)
	cmd.Flags().BoolVar(&disabled, "disabled", false, "Create the Automation disabled")
	cmd.Flags().BoolVar(&scheduleEnabled, "schedule-enabled", true, "Whether the schedule trigger is enabled")
	cmd.Flags().BoolVar(&httpPostTrigger, "http-post-trigger", false, "Create and enable an HTTP POST trigger")
	cmd.Flags().StringVar(&prompt, "prompt", "", "Task prompt sent to the AI SRE agent")
	cmd.Flags().StringVar(&promptFile, "prompt-file", "", "Read task prompt from a file, or - for stdin")
	cmd.Flags().StringVar(&environmentKind, "environment-kind", "", "Runtime environment kind: cloud or byoc; empty means automatic")
	cmd.Flags().StringVar(&environmentID, "environment-id", "", "BYOC Runner ID when --environment-kind=byoc")
	registerEnumFlag(cmd, "schedule", "hourly", "daily", "weekly", "cron")
	registerEnumFlag(cmd, "environment-kind", "cloud", "byoc")
	return cmd
}

func newAutomationListCmd() *cobra.Command {
	var (
		page    int
		limit   int
		scope   string
		keyword string
		enabled bool
		teamIDs []int64
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List visible Automations",
		Long:  curatedLong("List Automation rules visible to the caller.", "Automations", "RuleReadList"),
		Example: `  flashduty automation list --scope all --limit 20
  flashduty automation list --scope team --team-ids 123,456
  flashduty automation list --enabled=false --output-format json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				req := &flashduty.AutomationRuleListRequest{
					ListOptions: flashduty.ListOptions{Page: page, Limit: limit},
					Scope:       strings.TrimSpace(scope),
					Keyword:     strings.TrimSpace(keyword),
					TeamIDs:     teamIDs,
				}
				if cmd.Flags().Changed("enabled") {
					req.Enabled = flashduty.Bool(enabled)
				}
				out, _, err := ctx.Client.Automations.RuleReadList(cmdContext(ctx.Cmd), req)
				if err != nil {
					return err
				}
				return printGenericResult(ctx, out)
			})
		},
	}

	cmd.Flags().IntVar(&page, "page", 1, "Page number")
	cmd.Flags().IntVar(&limit, "limit", 20, "Page size, max 100")
	cmd.Flags().StringVar(&scope, "scope", "all", "Scope filter: all, personal, or team")
	cmd.Flags().StringVar(&keyword, "keyword", "", "Filter by name keyword")
	cmd.Flags().BoolVar(&enabled, "enabled", false, "Filter by enabled status")
	cmd.Flags().Int64SliceVar(&teamIDs, "team-ids", nil, "Filter to these team IDs; does not expand access")
	registerEnumFlag(cmd, "scope", "all", "personal", "team")
	return cmd
}

func newAutomationGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <rule_id>",
		Short: "Get an Automation",
		Long:  curatedLong("Get one Automation rule by ID.", "Automations", "RuleReadGet"),
		Args:  requireExactArg("rule_id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				out, _, err := ctx.Client.Automations.RuleReadGet(cmdContext(ctx.Cmd), &flashduty.AutomationRuleIDRequest{RuleID: ctx.Args[0]})
				if err != nil {
					return err
				}
				return printGenericResult(ctx, out)
			})
		},
	}
	return cmd
}

func newAutomationUpdateCmd() *cobra.Command {
	var (
		name                   string
		schedule               string
		at                     string
		weekday                string
		cronExpr               string
		enableRule             bool
		disableRule            bool
		enableSchedule         bool
		disableSchedule        bool
		prompt                 string
		promptFile             string
		environmentKind        string
		environmentID          string
		enableHTTPPostTrigger  bool
		disableHTTPPostTrigger bool
		rotateHTTPPostToken    bool
	)

	cmd := &cobra.Command{
		Use:   "update <rule_id>",
		Short: "Update an Automation",
		Long: curatedLong(`Update mutable fields on an Automation rule.

	The personal/team scope is intentionally not exposed here. Scope is immutable
	after creation; create a new Automation if the target person/team scope needs to change.

	Schedule helpers build a 5-field UTC cron expression. --at and --cron-expr are
	interpreted in UTC, not the caller's local timezone. Convert local wall-clock
	requests to UTC before passing --at or --cron-expr.`, "Automations", "RuleWriteUpdate"),
		Example: `  flashduty automation update auto_123 --name "Daily brief v2" --cron-expr "15 1 * * *"
  flashduty automation update auto_123 --disable
  flashduty automation update auto_123 --enable-http-post-trigger --rotate-http-post-token`,
		Args: requireExactArg("rule_id"),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if enableRule && disableRule {
				return fmt.Errorf("only one of --enable or --disable may be set")
			}
			if enableSchedule && disableSchedule {
				return fmt.Errorf("only one of --enable-schedule or --disable-schedule may be set")
			}
			if enableHTTPPostTrigger && disableHTTPPostTrigger {
				return fmt.Errorf("only one of --enable-http-post-trigger or --disable-http-post-trigger may be set")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				req := &flashduty.AutomationRuleUpdateRequest{RuleID: ctx.Args[0]}
				changed := false

				if cmd.Flags().Changed("name") {
					req.Name = flashduty.String(strings.TrimSpace(name))
					changed = true
				}
				if cmd.Flags().Changed("prompt") || cmd.Flags().Changed("prompt-file") {
					taskPrompt, err := resolveAutomationPrompt(cmd, prompt, promptFile)
					if err != nil {
						return err
					}
					req.Prompt = flashduty.String(taskPrompt)
					changed = true
				}
				if enableRule {
					req.Enabled = flashduty.Bool(true)
					changed = true
				}
				if disableRule {
					req.Enabled = flashduty.Bool(false)
					changed = true
				}
				if automationScheduleChanged(cmd) {
					cron, err := resolveAutomationCron(schedule, at, weekday, cronExpr)
					if err != nil {
						return err
					}
					req.CronExpr = flashduty.String(cron)
					changed = true
				}
				if enableSchedule {
					req.ScheduleTriggerEnabled = flashduty.Bool(true)
					changed = true
				}
				if disableSchedule {
					req.ScheduleTriggerEnabled = flashduty.Bool(false)
					changed = true
				}
				if cmd.Flags().Changed("environment-kind") {
					req.EnvironmentKind = flashduty.String(strings.TrimSpace(environmentKind))
					changed = true
				}
				if cmd.Flags().Changed("environment-id") {
					req.EnvironmentID = flashduty.String(strings.TrimSpace(environmentID))
					changed = true
				}
				if enableHTTPPostTrigger {
					req.HTTPPostTriggerEnabled = flashduty.Bool(true)
					changed = true
				}
				if disableHTTPPostTrigger {
					req.HTTPPostTriggerEnabled = flashduty.Bool(false)
					changed = true
				}
				if rotateHTTPPostToken {
					req.RotateHTTPPostTriggerToken = true
					changed = true
				}
				if !changed {
					return fmt.Errorf("at least one update field is required")
				}

				out, _, err := ctx.Client.Automations.RuleWriteUpdate(cmdContext(ctx.Cmd), req)
				if err != nil {
					return err
				}
				return printGenericResult(ctx, out)
			})
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "New Automation name")
	cmd.Flags().StringVar(&schedule, "schedule", "", "UTC schedule helper: hourly, daily, weekly, or cron")
	cmd.Flags().StringVar(&at, "at", "", "UTC time in HH:MM; for hourly schedules, only the minute is used. "+automationUTCNote)
	cmd.Flags().StringVar(&weekday, "weekday", "", "Weekday for weekly schedules: sun, mon, tue, wed, thu, fri, sat, or 0-7")
	cmd.Flags().StringVar(&cronExpr, "cron-expr", "", "Exact 5-field UTC cron expression; overrides --schedule helpers. "+automationUTCNote)
	cmd.Flags().BoolVar(&enableRule, "enable", false, "Enable the Automation")
	cmd.Flags().BoolVar(&disableRule, "disable", false, "Disable the Automation")
	cmd.Flags().BoolVar(&enableSchedule, "enable-schedule", false, "Enable the schedule trigger")
	cmd.Flags().BoolVar(&disableSchedule, "disable-schedule", false, "Disable the schedule trigger")
	cmd.Flags().StringVar(&prompt, "prompt", "", "New task prompt")
	cmd.Flags().StringVar(&promptFile, "prompt-file", "", "Read new task prompt from a file, or - for stdin")
	cmd.Flags().StringVar(&environmentKind, "environment-kind", "", "Runtime environment kind: cloud or byoc; empty means automatic")
	cmd.Flags().StringVar(&environmentID, "environment-id", "", "BYOC Runner ID when --environment-kind=byoc")
	cmd.Flags().BoolVar(&enableHTTPPostTrigger, "enable-http-post-trigger", false, "Enable or create the HTTP POST trigger")
	cmd.Flags().BoolVar(&disableHTTPPostTrigger, "disable-http-post-trigger", false, "Disable the HTTP POST trigger")
	cmd.Flags().BoolVar(&rotateHTTPPostToken, "rotate-http-post-token", false, "Rotate the HTTP POST trigger token")
	registerEnumFlag(cmd, "schedule", "hourly", "daily", "weekly", "cron")
	registerEnumFlag(cmd, "environment-kind", "cloud", "byoc")
	return cmd
}

func newAutomationDeleteCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete <rule_id>",
		Short: "Delete an Automation",
		Long: `Delete an Automation rule.

This is a destructive operation. Prompts for confirmation in an interactive
terminal unless --force is set. In non-interactive mode the command aborts
unless --force is provided.`,
		Args: requireExactArg("rule_id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				if !confirmAction(ctx.Cmd, fmt.Sprintf("Are you sure you want to delete Automation %s?", ctx.Args[0])) {
					_, _ = fmt.Fprintln(ctx.Writer, "Aborted.")
					return nil
				}
				_, _, err := ctx.Client.Automations.RuleWriteDelete(cmdContext(ctx.Cmd), &flashduty.AutomationRuleIDRequest{RuleID: ctx.Args[0]})
				if err != nil {
					return err
				}
				ctx.WriteResult(fmt.Sprintf("Deleted Automation %s.", ctx.Args[0]))
				return nil
			})
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	return cmd
}

func newAutomationRunsCmd() *cobra.Command {
	var (
		page        int
		limit       int
		since       string
		until       string
		status      string
		triggerKind string
	)

	cmd := &cobra.Command{
		Use:   "runs <rule_id>",
		Short: "List Automation runs",
		Long:  curatedLong("List run history for a rule the caller can manage.", "Automations", "RunReadList"),
		Args:  requireExactArg("rule_id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				req := &flashduty.AutomationRunListRequest{
					ListOptions: flashduty.ListOptions{Page: page, Limit: limit},
					RuleID:      ctx.Args[0],
					Status:      strings.TrimSpace(status),
					TriggerKind: strings.TrimSpace(triggerKind),
				}
				if v, ok, err := automationMillisFlag(cmd, "since", since); err != nil {
					return err
				} else if ok {
					req.StartedAfterMs = v
				}
				if v, ok, err := automationMillisFlag(cmd, "until", until); err != nil {
					return err
				} else if ok {
					req.StartedBeforeMs = v
				}
				out, _, err := ctx.Client.Automations.RunReadList(cmdContext(ctx.Cmd), req)
				if err != nil {
					return err
				}
				return printGenericResult(ctx, out)
			})
		},
	}

	cmd.Flags().IntVar(&page, "page", 1, "Page number")
	cmd.Flags().IntVar(&limit, "limit", 20, "Page size, max 100")
	cmd.Flags().StringVar(&since, "since", "", "Start-time lower bound; accepts duration, date, datetime, RFC3339, or unix seconds")
	cmd.Flags().StringVar(&until, "until", "", "Start-time upper bound; accepts duration, date, datetime, RFC3339, or unix seconds")
	cmd.Flags().StringVar(&status, "status", "", "Run status filter")
	cmd.Flags().StringVar(&triggerKind, "trigger-kind", "", "Trigger kind filter: schedule, debug, or http_post")
	registerEnumFlag(cmd, "status", "queued", "running", "retrying", "succeeded", "partial", "failed", "skipped", "abandoned")
	registerEnumFlag(cmd, "trigger-kind", "schedule", "debug", "http_post")
	return cmd
}

func newAutomationTemplatesCmd() *cobra.Command {
	var locale string

	cmd := &cobra.Command{
		Use:   "templates",
		Short: "List Automation templates",
		Long:  curatedLong("List preset Automation templates for the requested locale.", "Automations", "TemplateReadList"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				out, _, err := ctx.Client.Automations.TemplateReadList(cmdContext(ctx.Cmd), &flashduty.AutomationTemplateListRequest{Locale: strings.TrimSpace(locale)})
				if err != nil {
					return err
				}
				return printGenericResult(ctx, out)
			})
		},
	}

	cmd.Flags().StringVar(&locale, "locale", "", "Template locale such as zh-CN or en-US")
	return cmd
}

func newAutomationFireCmd() *cobra.Command {
	return buildAutomationFireCmd("fire <trigger_id>")
}

func newSafariAutomationTriggerFireCmd() *cobra.Command {
	return buildAutomationFireCmd("automation-triggers-{trigger_id}-fire <trigger_id>")
}

func buildAutomationFireCmd(use string) *cobra.Command {
	var (
		token    string
		text     string
		dedupKey string
		dataJSON string
	)

	cmd := &cobra.Command{
		Use:   use,
		Short: "Fire an Automation HTTP POST trigger",
		Long: `Trigger an Automation run through its HTTP POST trigger.

The trigger authenticates with its one-time token, not the account app key. Pass
--token or set FLASHDUTY_AUTOMATION_TRIGGER_TOKEN. Use --dedup-key to make
retries idempotent for the same trigger.`,
		Args: requireExactArg("trigger_id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				out, err := runAutomationFire(ctx, ctx.Args[0], token, text, dedupKey, dataJSON)
				if err != nil {
					return err
				}
				return printGenericResult(ctx, out)
			})
		},
	}

	cmd.Flags().StringVar(&token, "token", "", "HTTP POST trigger token; defaults to FLASHDUTY_AUTOMATION_TRIGGER_TOKEN")
	cmd.Flags().StringVar(&text, "text", "", "Context text passed to this run")
	cmd.Flags().StringVar(&dedupKey, "dedup-key", "", "Optional idempotency key")
	cmd.Flags().StringVar(&dataJSON, "data", "", "Full request body as JSON; typed flags override its fields. Accepts inline JSON, or - to read stdin.")
	return cmd
}

func runAutomationFire(ctx *RunContext, triggerID, token, text, dedupKey, dataJSON string) (*flashduty.AutomationFireAPITriggerResponse, error) {
	body, err := genAssembleBody(dataJSON, func(body map[string]any) error {
		if ctx.Cmd.Flags().Changed("text") {
			body["text"] = text
		}
		if ctx.Cmd.Flags().Changed("dedup-key") {
			body["dedup_key"] = dedupKey
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	req := new(flashduty.AutomationFireAPITriggerRequest)
	if err := genBindBody(body, req); err != nil {
		return nil, err
	}
	token = strings.TrimSpace(token)
	if token == "" {
		token = strings.TrimSpace(os.Getenv("FLASHDUTY_AUTOMATION_TRIGGER_TOKEN"))
	}
	if token == "" {
		return nil, fmt.Errorf("--token or FLASHDUTY_AUTOMATION_TRIGGER_TOKEN is required")
	}
	out, _, err := ctx.Client.Automations.TriggerWriteFire(cmdContext(ctx.Cmd), triggerID, token, req)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func attachSafariAutomationTriggerFire(root *cobra.Command) {
	safari := genGroup(root, "safari", "AI SRE API")
	genAddLeaf(safari, newSafariAutomationTriggerFireCmd())
}

func resolveAutomationPrompt(cmd *cobra.Command, prompt, promptFile string) (string, error) {
	promptChanged := cmd.Flags().Changed("prompt")
	fileChanged := cmd.Flags().Changed("prompt-file")
	if promptChanged && fileChanged {
		return "", fmt.Errorf("only one of --prompt or --prompt-file may be set")
	}
	if fileChanged {
		promptFile = strings.TrimSpace(promptFile)
		if promptFile == "" {
			return "", fmt.Errorf("--prompt-file must not be empty")
		}
		var (
			b   []byte
			err error
		)
		if promptFile == "-" {
			b, err = io.ReadAll(stdinReader)
		} else {
			b, err = os.ReadFile(promptFile)
		}
		if err != nil {
			return "", fmt.Errorf("failed to read prompt file: %w", err)
		}
		return strings.TrimSpace(string(b)), nil
	}
	return strings.TrimSpace(prompt), nil
}

func resolveAutomationCreateCron(cmd *cobra.Command, schedule, at, weekday, cronExpr string, httpPostTrigger bool, scheduleEnabled *bool) (string, error) {
	if httpPostTrigger && !automationScheduleChanged(cmd) && !cmd.Flags().Changed("schedule-enabled") {
		*scheduleEnabled = false
		return automationHTTPPostOnlyCron, nil
	}
	return resolveAutomationCron(schedule, at, weekday, cronExpr)
}

func automationScheduleChanged(cmd *cobra.Command) bool {
	return cmd.Flags().Changed("schedule") ||
		cmd.Flags().Changed("at") ||
		cmd.Flags().Changed("weekday") ||
		cmd.Flags().Changed("cron-expr")
}

func resolveAutomationCron(schedule, at, weekday, cronExpr string) (string, error) {
	cronExpr = strings.TrimSpace(cronExpr)
	if cronExpr != "" {
		return cronExpr, nil
	}

	schedule = strings.ToLower(strings.TrimSpace(schedule))
	if schedule == "" {
		schedule = "daily"
	}

	switch schedule {
	case "hourly":
		_, minute, err := parseAutomationAt(at, 0, 0)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%d * * * *", minute), nil
	case "daily":
		hour, minute, err := parseAutomationAt(at, 9, 0)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%d %d * * *", minute, hour), nil
	case "weekly":
		hour, minute, err := parseAutomationAt(at, 9, 0)
		if err != nil {
			return "", err
		}
		dow, err := parseAutomationWeekday(weekday)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%d %d * * %d", minute, hour, dow), nil
	case "cron":
		return "", fmt.Errorf("--cron-expr is required when --schedule=cron")
	default:
		return "", fmt.Errorf("invalid --schedule %q (want hourly, daily, weekly, or cron)", schedule)
	}
}

func parseAutomationAt(raw string, defaultHour, defaultMinute int) (int, int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultHour, defaultMinute, nil
	}
	parts := strings.Split(raw, ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("--at must be HH:MM")
	}
	hour, err := strconv.Atoi(parts[0])
	if err != nil || hour < 0 || hour > 23 {
		return 0, 0, fmt.Errorf("--at hour must be 0-23")
	}
	minute, err := strconv.Atoi(parts[1])
	if err != nil || minute < 0 || minute > 59 {
		return 0, 0, fmt.Errorf("--at minute must be 0-59")
	}
	return hour, minute, nil
}

func parseAutomationWeekday(raw string) (int, error) {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" {
		return 1, nil
	}
	names := map[string]int{
		"sun": 0, "sunday": 0,
		"mon": 1, "monday": 1,
		"tue": 2, "tuesday": 2,
		"wed": 3, "wednesday": 3,
		"thu": 4, "thursday": 4,
		"fri": 5, "friday": 5,
		"sat": 6, "saturday": 6,
	}
	if v, ok := names[raw]; ok {
		return v, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("--weekday must be sun-sat or 0-7")
	}
	if n < 0 || n > 7 {
		return 0, fmt.Errorf("--weekday must be sun-sat or 0-7")
	}
	if n == 7 {
		return 0, nil
	}
	return n, nil
}

func automationMillisFlag(cmd *cobra.Command, name, raw string) (int64, bool, error) {
	if !cmd.Flags().Changed(name) {
		return 0, false, nil
	}
	sec, err := timeutil.Parse(raw)
	if err != nil {
		return 0, false, fmt.Errorf("invalid --%s: %w", name, err)
	}
	return sec * 1000, true, nil
}
