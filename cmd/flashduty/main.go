package main

import (
	"fmt"
	"os"
	"runtime/debug"

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
	if err := cli.Execute(); err != nil {
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
			if s.Value == "true" {
				version += "-dirty"
			}
		}
	}
}
