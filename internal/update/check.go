package update

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	repoOwner            = "flashcatcloud"
	repoName             = "flashduty-cli"
	defaultUpdateBaseURL = "https://static.flashcat.cloud/flashduty-cli"
	checkInterval        = 24 * time.Hour
	httpTimeout          = 5 * time.Second
	stateFileName        = "state.yaml"
	maxResponseBytes     = 1 << 20 // 1MB
)

var autoHTTPTimeout = 2 * time.Second

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

func UpdateBaseURL() string {
	if v := strings.TrimSpace(os.Getenv("FLASHDUTY_UPDATE_BASE_URL")); v != "" {
		return strings.TrimRight(v, "/")
	}
	if v := strings.TrimSpace(os.Getenv("MIRROR_URL")); v != "" {
		return strings.TrimRight(v, "/")
	}
	return defaultUpdateBaseURL
}

func InstallShellURL() string      { return UpdateBaseURL() + "/install.sh" }
func InstallPowerShellURL() string { return UpdateBaseURL() + "/install.ps1" }

func InstallerEnv(base []string) []string {
	env := make([]string, 0, len(base)+1)
	for _, item := range base {
		if strings.HasPrefix(item, "MIRROR_URL=") {
			continue
		}
		env = append(env, item)
	}
	return append(env, "MIRROR_URL="+UpdateBaseURL())
}

func latestPointerURL() string {
	return UpdateBaseURL() + "/releases/latest"
}

func releasePageURL(tag string) string {
	return "https://github.com/" + repoOwner + "/" + repoName + "/releases/tag/" + tag
}

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
	return fetchLatestVersionWithTimeout(httpTimeout)
}

func fetchLatestVersionWithTimeout(timeout time.Duration) (string, string, error) {
	client := &http.Client{Timeout: timeout}
	resp, err := client.Get(latestPointerURL())
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch latest release: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("latest release endpoint returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return "", "", fmt.Errorf("failed to read latest release response: %w", err)
	}
	tag, err := parseLatestTag(string(body))
	if err != nil {
		return "", "", err
	}
	return tag, releasePageURL(tag), nil
}

func parseLatestTag(body string) (string, error) {
	line, _, _ := strings.Cut(body, "\n")
	tag := strings.TrimSpace(line)
	if tag == "" {
		return "", fmt.Errorf("empty latest release tag")
	}
	if len(tag) < 2 || tag[0] != 'v' || tag[1] < '0' || tag[1] > '9' {
		return "", fmt.Errorf("latest release tag is not valid: %q", tag)
	}
	for i := range len(tag) {
		ch := tag[i]
		if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') ||
			(ch >= '0' && ch <= '9') || ch == '.' || ch == '+' || ch == '-' {
			continue
		}
		return "", fmt.Errorf("latest release tag contains illegal characters: %q", tag)
	}
	return tag, nil
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
	return checkForUpdateWithTimeout(currentVersion, httpTimeout)
}

func CheckForUpdateAuto(currentVersion string) (*CheckResult, error) {
	result, err := checkForUpdateWithTimeout(currentVersion, autoHTTPTimeout)
	if err != nil {
		if IsTimeout(err) {
			_ = recordCheckAttempt()
		}
		return nil, err
	}
	return result, nil
}

func checkForUpdateWithTimeout(currentVersion string, timeout time.Duration) (*CheckResult, error) {
	tag, url, err := fetchLatestVersionWithTimeout(timeout)
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

func recordCheckAttempt() error {
	state := loadState()
	state.CheckedAt = time.Now()
	return saveState(state)
}

func IsTimeout(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func StateHasUpdate(currentVersion string) *CheckResult {
	if currentVersion == "dev" || currentVersion == "(devel)" {
		return nil
	}
	state := loadState()
	if state.LatestVersion == "" {
		return nil
	}
	if !IsNewer(state.LatestVersion, currentVersion) {
		return nil
	}
	return &CheckResult{
		CurrentVersion:  currentVersion,
		LatestVersion:   state.LatestVersion,
		LatestURL:       state.LatestURL,
		UpdateAvailable: true,
	}
}
