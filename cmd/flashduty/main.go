package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/flashcatcloud/flashduty-cli/internal/cli"
)

var (
	version = ""
	commit  = ""
	date    = ""
)

func main() {
	if version == "" {
		readBuildInfo()
	}
	if version == "" {
		version = "dev"
	}
	if commit == "" {
		commit = "none"
	}
	if date == "" {
		date = "unknown"
	}
	cli.SetVersionInfo(version, commit, date)
	// The CLI names itself after the command word the user typed, so a copy
	// installed under another name (install.sh INSTALLED_NAME) shows that name
	// in help, errors and completion scripts.
	name := strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe")
	if name == "" || name == "." || name == string(filepath.Separator) {
		name = "flashduty"
	}
	if err := cli.Execute(name); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

func readBuildInfo() {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	version = info.Main.Version
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			if len(s.Value) > 7 {
				commit = s.Value[:7]
			} else {
				commit = s.Value
			}
		case "vcs.time":
			date = s.Value
		case "vcs.modified":
			if s.Value == "true" && !strings.Contains(version, "dirty") {
				version += "-dirty"
			}
		}
	}
}
