package update

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

// setTestHome overrides the home directory for testing across all platforms.
func setTestHome(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("HOME", dir)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", dir)
	}
}

// clearCIEnv clears CI-related env vars so ShouldCheck doesn't short-circuit.
func clearCIEnv(t *testing.T) {
	t.Helper()
	t.Setenv("CI", "")
	t.Setenv("GITHUB_ACTIONS", "")
	t.Setenv("JENKINS_URL", "")
	t.Setenv("GITLAB_CI", "")
	t.Setenv("FLASHDUTY_NO_UPDATE_CHECK", "")
}

func TestStripV(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"v0.6.0", "0.6.0"},
		{"0.6.0", "0.6.0"},
		{"v1.0.0", "1.0.0"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := StripV(tt.in); got != tt.want {
			t.Errorf("StripV(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestCompareSemver(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"0.7.0", "0.6.0", 1},
		{"0.6.1", "0.6.0", 1},
		{"1.0.0", "0.9.9", 1},
		{"0.10.0", "0.9.0", 1},
		{"0.6.0", "0.7.0", -1},
		{"0.6.0", "0.6.0", 0},
		{"1.0.0", "1.0.0", 0},
		{"1.0.0-rc1", "1.0.0-rc2", 0},
		{"1.0.0-beta", "1.0.0", 0},
	}
	for _, tt := range tests {
		got := compareSemver(tt.a, tt.b)
		switch {
		case tt.want > 0 && got <= 0:
			t.Errorf("compareSemver(%q, %q) = %d, want >0", tt.a, tt.b, got)
		case tt.want < 0 && got >= 0:
			t.Errorf("compareSemver(%q, %q) = %d, want <0", tt.a, tt.b, got)
		case tt.want == 0 && got != 0:
			t.Errorf("compareSemver(%q, %q) = %d, want 0", tt.a, tt.b, got)
		}
	}
}

func TestIsNewer(t *testing.T) {
	tests := []struct {
		latest, current string
		want            bool
	}{
		{"v0.7.0", "0.6.0", true},
		{"v0.7.0", "v0.6.0", true},
		{"0.7.0", "0.6.0", true},
		{"v0.6.0", "0.6.0", false},
		{"v0.5.0", "0.6.0", false},
		{"v0.10.0", "0.9.0", true},
		{"v1.0.0-rc1", "0.9.0", true},
	}
	for _, tt := range tests {
		if got := IsNewer(tt.latest, tt.current); got != tt.want {
			t.Errorf("IsNewer(%q, %q) = %v, want %v", tt.latest, tt.current, got, tt.want)
		}
	}
}

func TestShouldCheck_DevVersion(t *testing.T) {
	if ShouldCheck("dev") {
		t.Error("ShouldCheck(\"dev\") = true, want false")
	}
	if ShouldCheck("(devel)") {
		t.Error("ShouldCheck(\"(devel)\") = true, want false")
	}
}

func TestShouldCheck_EnvDisabled(t *testing.T) {
	t.Setenv("FLASHDUTY_NO_UPDATE_CHECK", "1")
	if ShouldCheck("0.6.0") {
		t.Error("ShouldCheck should return false when FLASHDUTY_NO_UPDATE_CHECK=1")
	}
}

func TestShouldCheck_CI(t *testing.T) {
	t.Setenv("CI", "true")
	if ShouldCheck("0.6.0") {
		t.Error("ShouldCheck should return false in CI")
	}
}

func TestShouldCheck_RecentCheck(t *testing.T) {
	tmp := t.TempDir()
	setTestHome(t, tmp)
	clearCIEnv(t)

	dir := filepath.Join(tmp, ".flashduty")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}

	s := &State{CheckedAt: time.Now()}
	data, err := yaml.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, stateFileName), data, 0600); err != nil {
		t.Fatal(err)
	}

	if ShouldCheck("0.6.0") {
		t.Error("ShouldCheck should return false when checked recently")
	}
}

func TestShouldCheck_StaleCheck(t *testing.T) {
	tmp := t.TempDir()
	setTestHome(t, tmp)
	clearCIEnv(t)

	dir := filepath.Join(tmp, ".flashduty")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}

	s := &State{CheckedAt: time.Now().Add(-25 * time.Hour)}
	data, err := yaml.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, stateFileName), data, 0600); err != nil {
		t.Fatal(err)
	}

	if !ShouldCheck("0.6.0") {
		t.Error("ShouldCheck should return true when check is stale")
	}
}

func TestLoadSaveState(t *testing.T) {
	tmp := t.TempDir()
	setTestHome(t, tmp)

	now := time.Now().Truncate(time.Second)
	want := &State{
		CheckedAt:     now,
		LatestVersion: "v0.7.0",
		LatestURL:     "https://example.com/release",
	}

	if err := saveState(want); err != nil {
		t.Fatalf("saveState: %v", err)
	}

	got := loadState()
	if got.LatestVersion != want.LatestVersion {
		t.Errorf("LatestVersion = %q, want %q", got.LatestVersion, want.LatestVersion)
	}
	if got.LatestURL != want.LatestURL {
		t.Errorf("LatestURL = %q, want %q", got.LatestURL, want.LatestURL)
	}
	if got.CheckedAt.Unix() != want.CheckedAt.Unix() {
		t.Errorf("CheckedAt = %v, want %v", got.CheckedAt, want.CheckedAt)
	}
}

func TestLoadState_CorruptFile(t *testing.T) {
	tmp := t.TempDir()
	setTestHome(t, tmp)

	dir := filepath.Join(tmp, ".flashduty")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, stateFileName), []byte("{[invalid yaml"), 0600); err != nil {
		t.Fatal(err)
	}

	got := loadState()
	if !got.CheckedAt.IsZero() {
		t.Error("corrupt state file should return zero state")
	}
}

func TestFetchLatestVersion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		rel := githubRelease{
			TagName: "v0.7.0",
			HTMLURL: "https://github.com/flashcatcloud/flashduty-cli/releases/tag/v0.7.0",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(rel)
	}))
	defer srv.Close()

	origURL := apiURL
	apiURL = srv.URL
	defer func() { apiURL = origURL }()

	tag, url, err := fetchLatestVersion()
	if err != nil {
		t.Fatalf("fetchLatestVersion: %v", err)
	}
	if tag != "v0.7.0" {
		t.Errorf("tag = %q, want %q", tag, "v0.7.0")
	}
	if url != "https://github.com/flashcatcloud/flashduty-cli/releases/tag/v0.7.0" {
		t.Errorf("url = %q", url)
	}
}

func TestFetchLatestVersion_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	origURL := apiURL
	apiURL = srv.URL
	defer func() { apiURL = origURL }()

	_, _, err := fetchLatestVersion()
	if err == nil {
		t.Error("expected error for 404 response")
	}
}

func TestFetchLatestVersion_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{not valid json`))
	}))
	defer srv.Close()

	origURL := apiURL
	apiURL = srv.URL
	defer func() { apiURL = origURL }()

	_, _, err := fetchLatestVersion()
	if err == nil {
		t.Error("expected error for invalid JSON response")
	}
}

func TestCheckForUpdate(t *testing.T) {
	tmp := t.TempDir()
	setTestHome(t, tmp)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		rel := githubRelease{
			TagName: "v0.7.0",
			HTMLURL: "https://github.com/flashcatcloud/flashduty-cli/releases/tag/v0.7.0",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(rel)
	}))
	defer srv.Close()

	origURL := apiURL
	apiURL = srv.URL
	defer func() { apiURL = origURL }()

	result, err := CheckForUpdate("0.6.0")
	if err != nil {
		t.Fatalf("CheckForUpdate: %v", err)
	}
	if !result.UpdateAvailable {
		t.Error("UpdateAvailable = false, want true")
	}
	if result.LatestVersion != "v0.7.0" {
		t.Errorf("LatestVersion = %q, want %q", result.LatestVersion, "v0.7.0")
	}

	state := loadState()
	if state.LatestVersion != "v0.7.0" {
		t.Errorf("state.LatestVersion = %q, want %q", state.LatestVersion, "v0.7.0")
	}
}
