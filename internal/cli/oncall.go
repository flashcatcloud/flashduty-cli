package cli

import (
	"fmt"
	"strconv"
	"strings"

	flashduty "github.com/flashcatcloud/flashduty-sdk"
	"github.com/spf13/cobra"

	"github.com/flashcatcloud/flashduty-cli/internal/output"
	"github.com/flashcatcloud/flashduty-cli/internal/timeutil"
)

func newOncallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "oncall",
		Short: "Manage on-call schedules",
	}
	cmd.AddCommand(newOncallWhoCmd())
	cmd.AddCommand(newOncallScheduleCmd())
	return cmd
}

func newOncallScheduleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schedule",
		Short: "Manage schedules",
	}
	cmd.AddCommand(newOncallScheduleListCmd())
	cmd.AddCommand(newOncallScheduleGetCmd())
	return cmd
}

func newOncallWhoCmd() *cobra.Command {
	var query, team, since, until string
	var limit, page int

	cmd := &cobra.Command{
		Use:   "who",
		Short: "Show who is currently on call",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				startTime, err := timeutil.Parse(since)
				if err != nil {
					return fmt.Errorf("invalid --since: %w", err)
				}
				endTime, err := timeutil.Parse(until)
				if err != nil {
					return fmt.Errorf("invalid --until: %w", err)
				}

				input := &flashduty.ListSchedulesWithSlotsInput{
					Start: startTime,
					End:   endTime,
					Query: query,
					Limit: limit,
					Page:  page,
				}

				if team != "" {
					teamIDs, err := parseIntSlice(team)
					if err != nil {
						return fmt.Errorf("invalid --team: %w", err)
					}
					input.TeamIDs = teamIDs
				}

				result, err := ctx.Client.ListSchedulesWithSlots(cmdContext(ctx.Cmd), input)
				if err != nil {
					return err
				}

				cols := []output.Column{
					{Header: "SCHEDULE", MaxWidth: 30, Field: func(v any) string {
						s := v.(flashduty.ScheduleDetail)
						if s.ScheduleName != nil {
							return *s.ScheduleName
						}
						if s.Name != nil {
							return *s.Name
						}
						return "-"
					}},
					{Header: "ON_CALL", MaxWidth: 40, Field: func(v any) string {
						s := v.(flashduty.ScheduleDetail)
						return formatOncallMembers(s.CurOncall)
					}},
					{Header: "UNTIL", Field: func(v any) string {
						s := v.(flashduty.ScheduleDetail)
						if s.CurOncall != nil {
							return output.FormatTime(s.CurOncall.End)
						}
						return "-"
					}},
					{Header: "NEXT", MaxWidth: 40, Field: func(v any) string {
						s := v.(flashduty.ScheduleDetail)
						return formatOncallMembers(s.NextOncall)
					}},
				}

				return ctx.PrintTotal(result.Schedules, cols, int(result.Total))
			})
		},
	}

	cmd.Flags().StringVar(&query, "query", "", "Search by schedule name")
	cmd.Flags().StringVar(&team, "team", "", "Comma-separated team IDs")
	cmd.Flags().StringVar(&since, "since", "now", "Start of time range")
	cmd.Flags().StringVar(&until, "until", "+24h", "End of time range")
	cmd.Flags().IntVar(&limit, "limit", 20, "Max results")
	cmd.Flags().IntVar(&page, "page", 1, "Page number")

	return cmd
}

func newOncallScheduleListCmd() *cobra.Command {
	var query, team, since, until string
	var limit, page int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List schedules",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				startTime, err := timeutil.Parse(since)
				if err != nil {
					return fmt.Errorf("invalid --since: %w", err)
				}
				endTime, err := timeutil.Parse(until)
				if err != nil {
					return fmt.Errorf("invalid --until: %w", err)
				}

				input := &flashduty.ListSchedulesWithSlotsInput{
					Start: startTime,
					End:   endTime,
					Query: query,
					Limit: limit,
					Page:  page,
				}

				if team != "" {
					teamIDs, err := parseIntSlice(team)
					if err != nil {
						return fmt.Errorf("invalid --team: %w", err)
					}
					input.TeamIDs = teamIDs
				}

				result, err := ctx.Client.ListSchedulesWithSlots(cmdContext(ctx.Cmd), input)
				if err != nil {
					return err
				}

				cols := []output.Column{
					{Header: "ID", Field: func(v any) string {
						s := v.(flashduty.ScheduleDetail)
						return strconv.FormatInt(s.ScheduleID, 10)
					}},
					{Header: "NAME", MaxWidth: 30, Field: func(v any) string {
						s := v.(flashduty.ScheduleDetail)
						if s.ScheduleName != nil {
							return *s.ScheduleName
						}
						if s.Name != nil {
							return *s.Name
						}
						return "-"
					}},
					{Header: "STATUS", Field: func(v any) string {
						s := v.(flashduty.ScheduleDetail)
						if s.Disabled != nil && *s.Disabled != 0 {
							return "disabled"
						}
						return "enabled"
					}},
					{Header: "LAYERS", Field: func(v any) string {
						return scheduleLayerCount(v.(flashduty.ScheduleDetail))
					}},
				}

				return ctx.PrintTotal(result.Schedules, cols, int(result.Total))
			})
		},
	}

	cmd.Flags().StringVar(&query, "query", "", "Search by schedule name")
	cmd.Flags().StringVar(&team, "team", "", "Comma-separated team IDs")
	cmd.Flags().StringVar(&since, "since", "now", "Start of time range")
	cmd.Flags().StringVar(&until, "until", "+24h", "End of time range")
	cmd.Flags().IntVar(&limit, "limit", 20, "Max results")
	cmd.Flags().IntVar(&page, "page", 1, "Page number")

	return cmd
}

func newOncallScheduleGetCmd() *cobra.Command {
	var since, until string

	cmd := &cobra.Command{
		Use:   "get <schedule_id>",
		Short: "Get schedule detail",
		Args:  requireArgs("schedule_id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				scheduleID, err := strconv.ParseInt(ctx.Args[0], 10, 64)
				if err != nil {
					return fmt.Errorf("invalid schedule_id %q: %w", ctx.Args[0], err)
				}

				startTime, err := timeutil.Parse(since)
				if err != nil {
					return fmt.Errorf("invalid --since: %w", err)
				}
				endTime, err := timeutil.Parse(until)
				if err != nil {
					return fmt.Errorf("invalid --until: %w", err)
				}

				result, err := ctx.Client.GetScheduleDetail(cmdContext(ctx.Cmd), &flashduty.GetScheduleDetailInput{
					ScheduleID: scheduleID,
					Start:      startTime,
					End:        endTime,
				})
				if err != nil {
					return err
				}

				if ctx.JSON {
					return ctx.Printer.Print(result.Schedule, nil)
				}

				s := result.Schedule

				name := "-"
				if s.ScheduleName != nil {
					name = *s.ScheduleName
				} else if s.Name != nil {
					name = *s.Name
				}

				status := "enabled"
				if s.Disabled != nil && *s.Disabled != 0 {
					status = "disabled"
				}

				_, _ = fmt.Fprintf(ctx.Writer, "ID:            %d\n", s.ScheduleID)
				_, _ = fmt.Fprintf(ctx.Writer, "Name:          %s\n", name)
				_, _ = fmt.Fprintf(ctx.Writer, "Status:        %s\n", status)
				_, _ = fmt.Fprintf(ctx.Writer, "Layers:        %s\n", scheduleLayerCount(s))

				curOnCall := formatOncallMembers(s.CurOncall)
				curUntil := "-"
				if s.CurOncall != nil {
					curUntil = output.FormatTime(s.CurOncall.End)
				}
				_, _ = fmt.Fprintf(ctx.Writer, "Current:       %s (until %s)\n", curOnCall, curUntil)

				nextOnCall := formatOncallMembers(s.NextOncall)
				nextFrom := "-"
				if s.NextOncall != nil {
					nextFrom = output.FormatTime(s.NextOncall.Start)
				}
				_, _ = fmt.Fprintf(ctx.Writer, "Next:          %s (from %s)\n", nextOnCall, nextFrom)

				// Print computed slots table
				if len(s.FinalSchedule.Schedules) > 0 {
					_, _ = fmt.Fprintln(ctx.Writer, "")

					cols := []output.Column{
						{Header: "START", Field: func(v any) string {
							return output.FormatTime(v.(flashduty.ScheduleCalculatedSchedule).Start)
						}},
						{Header: "END", Field: func(v any) string {
							return output.FormatTime(v.(flashduty.ScheduleCalculatedSchedule).End)
						}},
						{Header: "GROUP", MaxWidth: 30, Field: func(v any) string {
							g := v.(flashduty.ScheduleCalculatedSchedule).Group
							if g.GroupName != "" {
								return g.GroupName
							}
							return g.Name
						}},
					}

					return ctx.Printer.Print(s.FinalSchedule.Schedules, cols)
				}

				return nil
			})
		},
	}

	cmd.Flags().StringVar(&since, "since", "now", "Start of time range")
	cmd.Flags().StringVar(&until, "until", "+7d", "End of time range")

	return cmd
}

// formatOncallMembers extracts member person IDs from a ScheduleOncallGroup and
// returns them as a comma-separated string. Since the schedule API returns person IDs
// (not names), we display IDs for now.
func scheduleLayerCount(s flashduty.ScheduleDetail) string {
	switch {
	case len(s.Layers) > 0:
		return fmt.Sprintf("%d", len(s.Layers))
	case len(s.ScheduleLayers) > 0:
		return fmt.Sprintf("%d", len(s.ScheduleLayers))
	case len(s.LayerSchedules) > 0:
		return fmt.Sprintf("%d", len(s.LayerSchedules))
	default:
		return "-"
	}
}

func formatOncallMembers(oncall *flashduty.ScheduleOncallGroup) string {
	if oncall == nil {
		return "-"
	}
	var ids []string
	for _, m := range oncall.Group.Members {
		for _, pid := range m.PersonIDs {
			ids = append(ids, strconv.FormatInt(pid, 10))
		}
	}
	if len(ids) == 0 {
		name := oncall.Group.GroupName
		if name == "" {
			name = oncall.Group.Name
		}
		if name != "" {
			return name
		}
		return "-"
	}
	return strings.Join(ids, ", ")
}
