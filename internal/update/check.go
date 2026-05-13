package update

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	repoOwner     = "flashcatcloud"
	repoName      = "flashduty-cli"
	checkInterval = 24 * time.Hour
	httpTimeout   = 5 * time.Second
	stateFileName = "state.yaml"
	installShURL  = "https://raw.githubusercontent.com/" + repoOwner + "/" + repoName + "/main/install.sh"
	installPs1URL = "https://raw.githubusercontent.com/" + repoOwner + "/" + repoName + "/main/install.ps1"
	maxResponseBytes = 1 << 20 // 1MB
)

var apiURL = "https://api.github.com/repos/" + repoOwner + "/" + repoName + "/releases/latest"

type State struct {
	CheckedAt     time.Time `yaml:"checked_at"`
	LatestVersion string    `yaml:"latest_version"`
	LatestURL     string    `yaml:"latest_url"`
}

type CheckResult struct {
	CurrentVersion  string
	LatestVersion   string
	LatestURL       string
	UpdateAvailable bool
}

type githubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

func InstallShellURL() string      { return installShURL }
func InstallPowerShellURL() string { return installPs1URL }

func stateDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to determine home directory: %w", err)
	}
	return filepath.Join(home, ".flashduty"), nil
}

func statePath() (string, error) {
	dir, err := stateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, stateFileName), nil
}

func loadState() *State {
	path, err := statePath()
	if err != nil {
		return &State{}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return &State{}
	}
	var s State
	if err := yaml.Unmarshal(data, &s); err != nil {
		return &State{}
	}
	return &s
}

func saveState(s *State) error {
	dir, err := stateDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}
	data, err := yaml.Marshal(s)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}
	path, err := statePath()
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func fetchLatestVersion() (string, string, error) {
	client := &http.Client{Timeout: httpTimeout}
	resp, err := client.Get(apiURL)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch latest release: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var rel githubRelease
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes)).Decode(&rel); err != nil {
		return "", "", fmt.Errorf("failed to parse release response: %w", err)
	}
	if rel.TagName == "" {
		return "", "", fmt.Errorf("empty tag_name in response")
	}
	return rel.TagName, rel.HTMLURL, nil
}

func StripV(v string) string {
	return strings.TrimPrefix(v, "v")
}

// stripPreRelease removes pre-release suffix (e.g. "1.0.0-rc1" -> "1.0.0").
func stripPreRelease(v string) string {
	if base, _, ok := strings.Cut(v, "-"); ok {
		return base
	}
	return v
}

func compareSemver(a, b string) int {
	a = stripPreRelease(a)
	b = stripPreRelease(b)
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")
	maxLen := max(len(aParts), len(bParts))
	for i := range maxLen {
		var ai, bi int
		if i < len(aParts) {
			ai, _ = strconv.Atoi(aParts[i])
		}
		if i < len(bParts) {
			bi, _ = strconv.Atoi(bParts[i])
		}
		if ai != bi {
			return ai - bi
		}
	}
	return 0
}

func IsNewer(latestTag, currentVersion string) bool {
	latest := StripV(latestTag)
	current := StripV(currentVersion)
	if latest == current {
		return false
	}
	return compareSemver(latest, current) > 0
}

func ShouldCheck(currentVersion string) bool {
	if currentVersion == "dev" || currentVersion == "(devel)" {
		return false
	}
	if os.Getenv("FLASHDUTY_NO_UPDATE_CHECK") == "1" {
		return false
	}
	if os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != "" ||
		os.Getenv("JENKINS_URL") != "" || os.Getenv("GITLAB_CI") != "" {
		return false
	}
	state := loadState()
	return time.Since(state.CheckedAt) >= checkInterval
}

func CheckForUpdate(currentVersion string) (*CheckResult, error) {
	tag, url, err := fetchLatestVersion()
	if err != nil {
		return nil, err
	}

	_ = saveState(&State{
		CheckedAt:     time.Now(),
		LatestVersion: tag,
		LatestURL:     url,
	})

	return &CheckResult{
		CurrentVersion:  currentVersion,
		LatestVersion:   tag,
		LatestURL:       url,
		UpdateAvailable: IsNewer(tag, currentVersion),
	}, nil
}
