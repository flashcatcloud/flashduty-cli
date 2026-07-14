package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// requireArgs returns a positional argument validator that produces descriptive
// error messages naming the missing arguments, e.g.:
//
//	"missing incident_id. Usage: flashduty incident update <id>"
func requireArgs(argNames ...string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) < len(argNames) {
			missing := argNames[len(args):]
			return fmt.Errorf("missing %s. Usage: %s", strings.Join(missing, ", "), cmd.UseLine())
		}
		return nil
	}
}

// requireBodyFieldOrArgs validates a required variadic positional that folds
// into a request-body field. The same field can also come from --data or from
// its typed flag, so zero positional args are valid when either source is set.
func requireBodyFieldOrArgs(name, flagName string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 || bodyFieldSourceSet(cmd, flagName) {
			return nil
		}
		return fmt.Errorf("missing %s. Usage: %s", name, cmd.UseLine())
	}
}

// requireBodyFieldOrExactArg validates a required scalar positional that folds
// into a request-body field. The value may instead come from --data or from the
// typed flag, but extra positionals are still rejected.
func requireBodyFieldOrExactArg(name, flagName string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) > 1 {
			return fmt.Errorf("expects exactly one %s. Usage: %s", name, cmd.UseLine())
		}
		if len(args) == 1 || bodyFieldSourceSet(cmd, flagName) {
			return nil
		}
		return fmt.Errorf("missing %s. Usage: %s", name, cmd.UseLine())
	}
}

func bodyFieldSourceSet(cmd *cobra.Command, flagName string) bool {
	return cmd.Flags().Changed("data") || cmd.Flags().Changed(flagName)
}

// optionalArg returns a positional argument validator that accepts zero or one
// argument named name. It backs generated commands whose positional folds into
// an OPTIONAL body field because the operation also accepts an alternative
// lookup key via a flag (e.g. `incident info` takes either the <incident-id>
// positional or --num). Extra arguments are rejected rather than silently
// dropped:
//
//   - zero or one arg: ok
//   - >1 args:         "expects at most one <name>. Usage: ..."
func optionalArg(name string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) > 1 {
			return fmt.Errorf("expects at most one %s. Usage: %s", name, cmd.UseLine())
		}
		return nil
	}
}

// requireExactlyOneFlag validates that exactly one of the named flags is set.
func requireExactlyOneFlag(cmd *cobra.Command, flagNames ...string) error {
	set := 0
	for _, name := range flagNames {
		if cmd.Flags().Changed(name) {
			set++
		}
	}
	if set != 1 {
		return fmt.Errorf("exactly one of --%s must be specified", strings.Join(flagNames, ", --"))
	}
	return nil
}

// confirmAction prompts the user for confirmation in interactive terminals.
// Returns true if the user confirms, or if running in non-interactive /
// structured-output (JSON/TOON) / --force mode.
func confirmAction(cmd *cobra.Command, message string) bool {
	if currentOutputFormat().Structured() {
		return true
	}
	force, _ := cmd.Flags().GetBool("force")
	if force {
		return true
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return false
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s [y/N]: ", message)
	scanner := bufio.NewScanner(cmd.InOrStdin())
	if scanner.Scan() {
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		return answer == "y" || answer == "yes"
	}
	return false
}
