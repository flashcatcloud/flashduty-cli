package cli

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/flashcatcloud/flashduty-cli/internal/update"
)

func newUpdateCmd() *cobra.Command {
	var flagCheck bool

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update flashduty to the latest version",
		RunE: func(cmd *cobra.Command, _ []string) error {
			w := cmd.OutOrStdout()
			_, _ = fmt.Fprintf(w, "Current version: %s\n", versionStr)
			_, _ = fmt.Fprintf(w, "Checking for updates...\n")

			result, err := update.CheckForUpdate(versionStr)
			if err != nil {
				return fmt.Errorf("failed to check for updates: %w", err)
			}

			if !result.UpdateAvailable {
				_, _ = fmt.Fprintf(w, "Already up to date (%s).\n", versionStr)
				return nil
			}

			_, _ = fmt.Fprintf(w, "A new version is available: v%s -> %s\n",
				update.StripV(versionStr), result.LatestVersion)
			_, _ = fmt.Fprintf(w, "Release: %s\n", result.LatestURL)

			if flagCheck {
				return nil
			}

			_, _ = fmt.Fprintf(w, "\nUpdating...\n")
			return runInstaller(cmd)
		},
	}

	cmd.Flags().BoolVar(&flagCheck, "check", false, "Only check for updates, do not install")
	return cmd
}

func runInstaller(cmd *cobra.Command) error {
	name, args := installerCommandSpec(runtime.GOOS, update.InstallShellURL(), update.InstallPowerShellURL())
	c := exec.Command(name, args...)

	c.Stdout = cmd.OutOrStdout()
	c.Stderr = cmd.ErrOrStderr()
	c.Stdin = os.Stdin
	c.Env = update.InstallerEnv(os.Environ())

	if err := c.Run(); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nUpdate complete. Run 'flashduty version' to verify.\n")
	return nil
}

func installerCommandSpec(goos, shellURL, powerShellURL string) (string, []string) {
	if goos == "windows" {
		return "powershell", []string{
			"-ExecutionPolicy",
			"Bypass",
			"-Command",
			"$u = $args[0]; irm $u | iex",
			powerShellURL,
		}
	}
	return "sh", []string{
		"-c",
		`curl -fsSL "$1" | sh`,
		"flashduty-installer",
		shellURL,
	}
}
