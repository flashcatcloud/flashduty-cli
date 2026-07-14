package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

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
			executable, err := os.Executable()
			if err != nil {
				return fmt.Errorf("failed to locate active executable: %w", err)
			}
			return runInstaller(cmd, executable, result.LatestVersion)
		},
	}

	cmd.Flags().BoolVar(&flagCheck, "check", false, "Only check for updates, do not install")
	return cmd
}

func runInstaller(cmd *cobra.Command, executable, expectedVersion string) error {
	isWindows := runtime.GOOS == "windows"
	name, args := installerCommandSpec(runtime.GOOS, update.InstallShellURL(), update.InstallPowerShellURL())
	c := exec.Command(name, args...)

	c.Stdout = cmd.OutOrStdout()
	c.Stderr = cmd.ErrOrStderr()
	c.Stdin = os.Stdin
	installedName := "flashduty"
	c.Env = update.InstallerEnv(os.Environ())
	if !isWindows {
		installedName = filepath.Base(executable)
		env := c.Env[:0]
		for _, item := range c.Env {
			if strings.HasPrefix(item, "FLASHDUTY_INSTALL_DIR=") || strings.HasPrefix(item, "INSTALLED_NAME=") {
				continue
			}
			env = append(env, item)
		}
		c.Env = append(env,
			"FLASHDUTY_INSTALL_DIR="+filepath.Dir(executable),
			"INSTALLED_NAME="+installedName,
		)
	}

	if err := c.Run(); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	if !isWindows {
		out, err := exec.Command(executable, "version", "--json").Output()
		if err != nil {
			return fmt.Errorf("update installed but active executable %s could not be verified: %w", executable, err)
		}
		var info struct {
			Version string `json:"version"`
		}
		if err := json.Unmarshal(out, &info); err != nil {
			return fmt.Errorf("update installed but active executable %s returned unparseable version output: %w", executable, err)
		}
		if update.StripV(info.Version) != update.StripV(expectedVersion) {
			return fmt.Errorf("update installed but active executable %s is still stale: expected %s, got %s", executable, expectedVersion, info.Version)
		}
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nUpdate complete. Run '%s version' to verify.\n", installedName)
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
