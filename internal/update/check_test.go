package update

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("v0.7.0\n"))
	}))
	defer srv.Close()

	t.Setenv("FLASHDUTY_UPDATE_BASE_URL", srv.URL)
	t.Setenv("MIRROR_URL", "")

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

func TestFetchLatestVersion_FromCDNLatestPointer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/releases/latest" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("v1.2.3\n"))
	}))
	defer srv.Close()

	t.Setenv("FLASHDUTY_UPDATE_BASE_URL", srv.URL+"/")
	t.Setenv("MIRROR_URL", "")

	tag, url, err := fetchLatestVersion()
	if err != nil {
		t.Fatalf("fetchLatestVersion: %v", err)
	}
	if tag != "v1.2.3" {
		t.Errorf("tag = %q, want %q", tag, "v1.2.3")
	}
	if url != "https://github.com/flashcatcloud/flashduty-cli/releases/tag/v1.2.3" {
		t.Errorf("url = %q", url)
	}
}

func TestFetchLatestVersion_RejectsInvalidLatestPointer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("../bad\n"))
	}))
	defer srv.Close()

	t.Setenv("FLASHDUTY_UPDATE_BASE_URL", srv.URL)
	t.Setenv("MIRROR_URL", "")

	_, _, err := fetchLatestVersion()
	if err == nil {
		t.Fatal("expected invalid latest pointer to fail")
	}
}

func TestUpdateBaseURLAndInstallerURLs(t *testing.T) {
	t.Setenv("FLASHDUTY_UPDATE_BASE_URL", "")
	t.Setenv("MIRROR_URL", "")
	if got := UpdateBaseURL(); got != defaultUpdateBaseURL {
		t.Fatalf("UpdateBaseURL() = %q, want %q", got, defaultUpdateBaseURL)
	}
	if got := InstallShellURL(); got != "https://static.flashcat.cloud/flashduty-cli/install.sh" {
		t.Fatalf("InstallShellURL() = %q", got)
	}
	if got := InstallPowerShellURL(); got != "https://static.flashcat.cloud/flashduty-cli/install.ps1" {
		t.Fatalf("InstallPowerShellURL() = %q", got)
	}

	t.Setenv("FLASHDUTY_UPDATE_BASE_URL", "https://mirror.example.com/fduty/")
	if got := UpdateBaseURL(); got != "https://mirror.example.com/fduty" {
		t.Fatalf("UpdateBaseURL() override = %q", got)
	}
	if got := InstallShellURL(); got != "https://mirror.example.com/fduty/install.sh" {
		t.Fatalf("InstallShellURL() override = %q", got)
	}
	if got := InstallPowerShellURL(); got != "https://mirror.example.com/fduty/install.ps1" {
		t.Fatalf("InstallPowerShellURL() override = %q", got)
	}
}

func TestInstallerEnvPassesUpdateBaseAsMirrorURL(t *testing.T) {
	t.Setenv("FLASHDUTY_UPDATE_BASE_URL", "https://mirror.example.com/fduty/")
	t.Setenv("MIRROR_URL", "")

	env := InstallerEnv([]string{"PATH=/bin", "MIRROR_URL=https://old.example.com"})
	want := "MIRROR_URL=https://mirror.example.com/fduty"
	found := 0
	for _, item := range env {
		if strings.HasPrefix(item, "MIRROR_URL=") {
			found++
			if item != want {
				t.Fatalf("MIRROR_URL entry = %q, want %q", item, want)
			}
		}
	}
	if found != 1 {
		t.Fatalf("found %d MIRROR_URL entries, want 1 in %#v", found, env)
	}
}

func TestFetchLatestVersion_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	t.Setenv("FLASHDUTY_UPDATE_BASE_URL", srv.URL)
	t.Setenv("MIRROR_URL", "")

	_, _, err := fetchLatestVersion()
	if err == nil {
		t.Error("expected error for 404 response")
	}
}

func TestFetchLatestVersion_EmptyLatestPointer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("\n"))
	}))
	defer srv.Close()

	t.Setenv("FLASHDUTY_UPDATE_BASE_URL", srv.URL)
	t.Setenv("MIRROR_URL", "")

	_, _, err := fetchLatestVersion()
	if err == nil {
		t.Error("expected error for empty latest pointer")
	}
}

func TestCheckForUpdate(t *testing.T) {
	tmp := t.TempDir()
	setTestHome(t, tmp)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("v0.7.0\n"))
	}))
	defer srv.Close()

	t.Setenv("FLASHDUTY_UPDATE_BASE_URL", srv.URL)
	t.Setenv("MIRROR_URL", "")

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

func TestCheckForUpdateAuto_RecordsAttemptOnTimeout(t *testing.T) {
	tmp := t.TempDir()
	setTestHome(t, tmp)
	clearCIEnv(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(50 * time.Millisecond)
		_, _ = w.Write([]byte("v0.7.0\n"))
	}))
	defer srv.Close()

	t.Setenv("FLASHDUTY_UPDATE_BASE_URL", srv.URL)
	t.Setenv("MIRROR_URL", "")

	origTimeout := autoHTTPTimeout
	autoHTTPTimeout = time.Nanosecond
	defer func() { autoHTTPTimeout = origTimeout }()

	_, err := CheckForUpdateAuto("0.6.0")
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !IsTimeout(err) {
		t.Fatalf("IsTimeout(%v) = false, want true", err)
	}
	if ShouldCheck("0.6.0") {
		t.Fatal("ShouldCheck should be false after an auto-check timeout records today's attempt")
	}
	state := loadState()
	if time.Since(state.CheckedAt) > time.Minute {
		t.Fatalf("CheckedAt was not refreshed after timeout: %v", state.CheckedAt)
	}
}

func TestCheckForUpdateAuto_DoesNotRecordAttemptOnNonTimeoutError(t *testing.T) {
	tmp := t.TempDir()
	setTestHome(t, tmp)
	clearCIEnv(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	t.Setenv("FLASHDUTY_UPDATE_BASE_URL", srv.URL)
	t.Setenv("MIRROR_URL", "")

	_, err := CheckForUpdateAuto("0.6.0")
	if err == nil {
		t.Fatal("expected error")
	}
	if IsTimeout(err) {
		t.Fatalf("IsTimeout(%v) = true, want false", err)
	}
	if !ShouldCheck("0.6.0") {
		t.Fatal("ShouldCheck should stay true after a non-timeout auto-check error")
	}
}

func TestStateHasUpdate(t *testing.T) {
	tmp := t.TempDir()
	setTestHome(t, tmp)

	if got := StateHasUpdate("0.6.0"); got != nil {
		t.Error("StateHasUpdate should return nil when no state file exists")
	}

	_ = saveState(&State{
		CheckedAt:     time.Now(),
		LatestVersion: "v0.7.0",
		LatestURL:     "https://example.com/v0.7.0",
	})

	got := StateHasUpdate("0.6.0")
	if got == nil {
		t.Fatal("StateHasUpdate should return non-nil when update is available")
	}
	if !got.UpdateAvailable {
		t.Error("UpdateAvailable = false, want true")
	}
	if got.LatestVersion != "v0.7.0" {
		t.Errorf("LatestVersion = %q, want %q", got.LatestVersion, "v0.7.0")
	}
}

func TestStateHasUpdate_AlreadyCurrent(t *testing.T) {
	tmp := t.TempDir()
	setTestHome(t, tmp)

	_ = saveState(&State{
		CheckedAt:     time.Now(),
		LatestVersion: "v0.6.0",
		LatestURL:     "https://example.com/v0.6.0",
	})

	if got := StateHasUpdate("0.6.0"); got != nil {
		t.Error("StateHasUpdate should return nil when already up to date")
	}
}

func TestStateHasUpdate_DevVersion(t *testing.T) {
	tmp := t.TempDir()
	setTestHome(t, tmp)

	_ = saveState(&State{
		CheckedAt:     time.Now(),
		LatestVersion: "v1.0.0",
	})

	if got := StateHasUpdate("dev"); got != nil {
		t.Error("StateHasUpdate should return nil for dev version")
	}
	if got := StateHasUpdate("(devel)"); got != nil {
		t.Error("StateHasUpdate should return nil for (devel) version")
	}
}
