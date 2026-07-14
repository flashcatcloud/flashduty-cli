package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func writeExecutable(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestRunInstallerUpdatesInvokedBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell installer test")
	}

	dir := t.TempDir()
	executable := filepath.Join(dir, ".flashduty", "bin", "fduty")
	writeExecutable(t, executable, `#!/bin/sh
if [ "$1" = "version" ] && [ "$2" = "--json" ]; then
  echo '{"version":"1.3.6"}'
else
  echo 'flashduty version 1.3.6 (old) built old'
fi
`)

	fakeBin := filepath.Join(dir, "fake-bin")
	writeExecutable(t, filepath.Join(fakeBin, "sh"), `#!/bin/sh
cat > "$FLASHDUTY_INSTALL_DIR/$INSTALLED_NAME" <<'EOF'
#!/bin/sh
if [ "$1" = "version" ] && [ "$2" = "--json" ]; then
  echo '{"version":"1.3.24"}'
else
  echo 'flashduty version 1.3.24 (new) built now'
fi
EOF
chmod +x "$FLASHDUTY_INSTALL_DIR/$INSTALLED_NAME"
`)
	t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("FLASHDUTY_INSTALL_DIR", filepath.Join(dir, "wrong-dir"))
	t.Setenv("INSTALLED_NAME", "wrong-name")

	var stdout, stderr bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	if err := runInstaller(cmd, executable, "v1.3.24"); err != nil {
		t.Fatalf("runInstaller: %v; stderr=%s", err, stderr.String())
	}

	out, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "version 1.3.24") {
		t.Fatalf("active executable was not replaced: %s", out)
	}
	if !strings.Contains(stdout.String(), "Update complete") {
		t.Fatalf("stdout = %q, want update success", stdout.String())
	}
}

func TestRunInstallerRejectsShadowedInstall(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell installer test")
	}

	dir := t.TempDir()
	executable := filepath.Join(dir, ".flashduty", "bin", "fduty")
	writeExecutable(t, executable, `#!/bin/sh
if [ "$1" = "version" ] && [ "$2" = "--json" ]; then
  echo '{"version":"1.3.6"}'
else
  echo 'flashduty version 1.3.6 (old) built old'
fi
`)

	fakeBin := filepath.Join(dir, "fake-bin")
	writeExecutable(t, filepath.Join(fakeBin, "sh"), "#!/bin/sh\nexit 0\n")
	t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))

	var stdout bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&stdout)
	err := runInstaller(cmd, executable, "v1.3.24")
	if err == nil || !strings.Contains(err.Error(), executable) || !strings.Contains(err.Error(), "1.3.6") {
		t.Fatalf("runInstaller error = %v, want active path and stale version", err)
	}
	if strings.Contains(stdout.String(), "Update complete") {
		t.Fatalf("reported success for stale executable: %q", stdout.String())
	}
}

func TestInstallerCommandSpecPassesInstallerURLAsArgument(t *testing.T) {
	shellURL := `https://mirror.example.com/fduty/install.sh; echo injected`
	psURL := `https://mirror.example.com/fduty/install.ps1; Write-Host injected`

	name, args := installerCommandSpec("linux", shellURL, psURL)
	if name != "sh" {
		t.Fatalf("unix installer command = %q, want sh", name)
	}
	if len(args) == 0 || args[len(args)-1] != shellURL {
		t.Fatalf("unix installer URL should be passed as the last argument, got %#v", args)
	}
	if strings.Contains(strings.Join(args[:len(args)-1], " "), "mirror.example.com") {
		t.Fatalf("unix installer URL was interpolated into shell command args: %#v", args)
	}

	name, args = installerCommandSpec("windows", shellURL, psURL)
	if name != "powershell" {
		t.Fatalf("windows installer command = %q, want powershell", name)
	}
	if !slices.Contains(args, psURL) {
		t.Fatalf("windows installer URL should be passed as an argument, got %#v", args)
	}
	for _, arg := range args {
		if arg != psURL && strings.Contains(arg, "mirror.example.com") {
			t.Fatalf("windows installer URL was interpolated into PowerShell command: %#v", args)
		}
	}
}
