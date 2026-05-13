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
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "Current version: %s\n", versionStr)
			fmt.Fprintf(cmd.OutOrStdout(), "Checking for updates...\n")

			result, err := update.CheckForUpdate(versionStr)
			if err != nil {
				return fmt.Errorf("failed to check for updates: %w", err)
			}

			if !result.UpdateAvailable {
				fmt.Fprintf(cmd.OutOrStdout(), "Already up to date (%s).\n", versionStr)
				return nil
			}

			fmt.Fprintf(cmd.OutOrStdout(), "A new version is available: v%s -> %s\n",
				update.StripV(versionStr), result.LatestVersion)
			fmt.Fprintf(cmd.OutOrStdout(), "Release: %s\n", result.LatestURL)

			if flagCheck {
				return nil
			}

			fmt.Fprintf(cmd.OutOrStdout(), "\nUpdating...\n")
			return runInstaller(cmd)
		},
	}

	cmd.Flags().BoolVar(&flagCheck, "check", false, "Only check for updates, do not install")
	return cmd
}

func runInstaller(cmd *cobra.Command) error {
	var c *exec.Cmd
	if runtime.GOOS == "windows" {
		c = exec.Command("powershell", "-Command",
			fmt.Sprintf("irm %s | iex", update.InstallPowerShellURL()))
	} else {
		c = exec.Command("sh", "-c",
			fmt.Sprintf("curl -fsSL %s | sh", update.InstallShellURL()))
	}

	c.Stdout = cmd.OutOrStdout()
	c.Stderr = cmd.ErrOrStderr()
	c.Stdin = os.Stdin

	if err := c.Run(); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\nUpdate complete. Run 'flashduty version' to verify.\n")
	return nil
}
