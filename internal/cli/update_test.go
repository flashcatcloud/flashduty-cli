package cli

import (
	"slices"
	"strings"
	"testing"
)

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
