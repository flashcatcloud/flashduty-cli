package cli

import (
	"fmt"
	"strconv"
	"strings"

	gflashduty "github.com/flashcatcloud/go-flashduty"
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
			return runGFCommand(cmd, args, func(ctx *RunContext) error {
				startTime, err := timeutil.Parse(since)
				if err != nil {
					return fmt.Errorf("invalid --since: %w", err)
				}
				endTime, err := timeutil.Parse(until)
				if err != nil {
					return fmt.Errorf("invalid --until: %w", err)
				}

				req := &gflashduty.ScheduleListRequest{
					Start: startTime,
					End:   endTime,
					Query: query,
				}
				req.Limit = limit
				req.Page = page

				if team != "" {
					teamIDs, err := parseIntSlice(team)
					if err != nil {
						return fmt.Errorf("invalid --team: %w", err)
					}
					req.TeamIDs = teamIDs
				}

				result, _, err := ctx.GFClient.Schedules.List(cmdContext(ctx.Cmd), req)
				if err != nil {
					return err
				}

				// Resolve on-call person IDs to display names (best-effort).
				nameByID := resolveScheduleOncallPeople(ctx, result.Items)

				cols := []output.Column{
					{Header: "SCHEDULE", MaxWidth: 30, Field: func(v any) string {
						return scheduleDisplayName(v.(gflashduty.ScheduleItem))
					}},
					{Header: "ON_CALL", MaxWidth: 40, Field: func(v any) string {
						s := v.(gflashduty.ScheduleItem)
						return formatOncallMembers(&s.CurOncall, nameByID)
					}},
					{Header: "UNTIL", Field: func(v any) string {
						return output.FormatTime(v.(gflashduty.ScheduleItem).CurOncall.End)
					}},
					{Header: "NEXT", MaxWidth: 40, Field: func(v any) string {
						s := v.(gflashduty.ScheduleItem)
						return formatOncallMembers(&s.NextOncall, nameByID)
					}},
				}

				return ctx.PrintTotal(result.Items, cols, int(result.Total))
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
			return runGFCommand(cmd, args, func(ctx *RunContext) error {
				startTime, err := timeutil.Parse(since)
				if err != nil {
					return fmt.Errorf("invalid --since: %w", err)
				}
				endTime, err := timeutil.Parse(until)
				if err != nil {
					return fmt.Errorf("invalid --until: %w", err)
				}

				req := &gflashduty.ScheduleListRequest{
					Start: startTime,
					End:   endTime,
					Query: query,
				}
				req.Limit = limit
				req.Page = page

				if team != "" {
					teamIDs, err := parseIntSlice(team)
					if err != nil {
						return fmt.Errorf("invalid --team: %w", err)
					}
					req.TeamIDs = teamIDs
				}

				result, _, err := ctx.GFClient.Schedules.List(cmdContext(ctx.Cmd), req)
				if err != nil {
					return err
				}

				cols := []output.Column{
					{Header: "ID", Field: func(v any) string {
						return strconv.FormatInt(scheduleID(v.(gflashduty.ScheduleItem)), 10)
					}},
					{Header: "NAME", MaxWidth: 30, Field: func(v any) string {
						return scheduleDisplayName(v.(gflashduty.ScheduleItem))
					}},
					{Header: "STATUS", Field: func(v any) string {
						s := v.(gflashduty.ScheduleItem)
						if s.Disabled != 0 {
							return "disabled"
						}
						return "enabled"
					}},
					{Header: "LAYERS", Field: func(v any) string {
						return scheduleLayerCount(v.(gflashduty.ScheduleItem))
					}},
				}

				return ctx.PrintTotal(result.Items, cols, int(result.Total))
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
			return runGFCommand(cmd, args, func(ctx *RunContext) error {
				scheduleIDArg, err := strconv.ParseInt(ctx.Args[0], 10, 64)
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

				s, _, err := ctx.GFClient.Schedules.Info(cmdContext(ctx.Cmd), &gflashduty.ScheduleInfoRequest{
					ScheduleID: scheduleIDArg,
					Start:      startTime,
					End:        endTime,
				})
				if err != nil {
					return err
				}

				if ctx.Structured() {
					return ctx.Printer.Print(s, nil)
				}

				// Resolve on-call person IDs to display names (best-effort).
				nameByID := resolveScheduleOncallPeople(ctx, []gflashduty.ScheduleItem{*s})

				status := "enabled"
				if s.Disabled != 0 {
					status = "disabled"
				}

				_, _ = fmt.Fprintf(ctx.Writer, "ID:            %d\n", scheduleID(*s))
				_, _ = fmt.Fprintf(ctx.Writer, "Name:          %s\n", scheduleDisplayName(*s))
				_, _ = fmt.Fprintf(ctx.Writer, "Status:        %s\n", status)
				_, _ = fmt.Fprintf(ctx.Writer, "Layers:        %s\n", scheduleLayerCount(*s))

				curOnCall := formatOncallMembers(&s.CurOncall, nameByID)
				curUntil := output.FormatTime(s.CurOncall.End)
				_, _ = fmt.Fprintf(ctx.Writer, "Current:       %s (until %s)\n", curOnCall, curUntil)

				nextOnCall := formatOncallMembers(&s.NextOncall, nameByID)
				nextFrom := output.FormatTime(s.NextOncall.Start)
				_, _ = fmt.Fprintf(ctx.Writer, "Next:          %s (from %s)\n", nextOnCall, nextFrom)

				// Print computed slots table
				if len(s.FinalSchedule.Schedules) > 0 {
					_, _ = fmt.Fprintln(ctx.Writer, "")

					cols := []output.Column{
						{Header: "START", Field: func(v any) string {
							return output.FormatTime(v.(gflashduty.ScheduleCalculatedSchedule).Start)
						}},
						{Header: "END", Field: func(v any) string {
							return output.FormatTime(v.(gflashduty.ScheduleCalculatedSchedule).End)
						}},
						{Header: "GROUP", MaxWidth: 30, Field: func(v any) string {
							g := v.(gflashduty.ScheduleCalculatedSchedule).Group
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

// scheduleID returns the schedule's numeric ID, preferring schedule_id and
// falling back to the legacy id field.
func scheduleID(s gflashduty.ScheduleItem) int64 {
	if s.ScheduleID != 0 {
		return s.ScheduleID
	}
	return s.ID
}

// scheduleDisplayName returns the schedule's display name, preferring
// schedule_name and falling back to the legacy name field.
func scheduleDisplayName(s gflashduty.ScheduleItem) string {
	if s.ScheduleName != "" {
		return s.ScheduleName
	}
	if s.Name != "" {
		return s.Name
	}
	return "-"
}

func scheduleLayerCount(s gflashduty.ScheduleItem) string {
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

// formatOncallMembers renders an on-call group's members as display names,
// resolving person IDs through nameByID (best-effort, falling back to the
// numeric ID), and finally to the group name when no members are present.
func formatOncallMembers(oncall *gflashduty.ScheduleOncallGroup, nameByID map[int64]string) string {
	if oncall == nil {
		return "-"
	}
	var names []string
	for _, m := range oncall.Group.Members {
		for _, pid := range m.PersonIDs {
			if n, ok := nameByID[pid]; ok && n != "" {
				names = append(names, n)
			} else {
				names = append(names, strconv.FormatInt(pid, 10))
			}
		}
	}
	if len(names) == 0 {
		name := oncall.Group.GroupName
		if name == "" {
			name = oncall.Group.Name
		}
		if name != "" {
			return name
		}
		return "-"
	}
	return strings.Join(names, ", ")
}

// resolveScheduleOncallPeople collects the on-call person IDs across the given
// schedules' current and next on-call groups and resolves them to display names
// via /person/infos, replicating the name lookup the legacy SDK fronted.
// Best-effort: a lookup failure yields a nil map and callers fall back to the
// numeric ID.
func resolveScheduleOncallPeople(rc *RunContext, items []gflashduty.ScheduleItem) map[int64]string {
	seen := make(map[int64]struct{})
	ids := make([]uint64, 0)
	collect := func(g gflashduty.ScheduleOncallGroup) {
		for _, m := range g.Group.Members {
			for _, pid := range m.PersonIDs {
				if pid == 0 {
					continue
				}
				if _, ok := seen[pid]; ok {
					continue
				}
				seen[pid] = struct{}{}
				ids = append(ids, uint64(pid))
			}
		}
	}
	for _, s := range items {
		collect(s.CurOncall)
		collect(s.NextOncall)
	}
	if len(ids) == 0 {
		return nil
	}
	resp, _, err := rc.GFClient.Members.PersonInfos(cmdContext(rc.Cmd), &gflashduty.PersonInfosRequest{PersonIDs: ids})
	if err != nil || resp == nil {
		return nil
	}
	out := make(map[int64]string, len(resp.Items))
	for _, p := range resp.Items {
		out[int64(p.PersonID)] = p.PersonName
	}
	return out
}
