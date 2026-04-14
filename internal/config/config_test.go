package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// setHomeDir overrides HOME (or USERPROFILE on Windows) so that
// ConfigDir/ConfigPath resolve to a test-local temp directory.
func setHomeDir(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", tmpDir)
	} else {
		t.Setenv("HOME", tmpDir)
	}
	return tmpDir
}

// clearEnvVars ensures FLASHDUTY_* env vars are unset for the test.
func clearEnvVars(t *testing.T) {
	t.Helper()
	t.Setenv("FLASHDUTY_APP_KEY", "")
	t.Setenv("FLASHDUTY_BASE_URL", "")
	// t.Setenv sets the value; setting to "" effectively clears it for
	// os.Getenv checks that test for v != "".
}

// --- Tests 47-51: MaskKey (table-driven) ---

func TestMaskKey(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want string
	}{
		{
			name: "long key",
			key:  "fd_abc123456789xyz",
			want: "fd_a...9xyz",
		},
		{
			name: "short key (<=8)",
			key:  "short",
			want: "****",
		},
		{
			name: "exactly 8 chars",
			key:  "12345678",
			want: "****",
		},
		{
			name: "exactly 9 chars",
			key:  "123456789",
			want: "1234...6789",
		},
		{
			name: "empty string",
			key:  "",
			want: "****",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MaskKey(tt.key)
			if got != tt.want {
				t.Errorf("MaskKey(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

// --- Test 52: Load defaults ---

func TestLoad_Defaults(t *testing.T) {
	setHomeDir(t)
	clearEnvVars(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	if cfg.BaseURL != DefaultBaseURL {
		t.Errorf("BaseURL = %q, want %q", cfg.BaseURL, DefaultBaseURL)
	}
	if cfg.AppKey != "" {
		t.Errorf("AppKey = %q, want empty string", cfg.AppKey)
	}
}

// --- Test 53: Load from file ---

func TestLoad_FromFile(t *testing.T) {
	tmpDir := setHomeDir(t)
	clearEnvVars(t)

	configDir := filepath.Join(tmpDir, configDirName)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	content := []byte("app_key: test-key-from-file\nbase_url: https://custom.example.com\n")
	if err := os.WriteFile(filepath.Join(configDir, configFileName), content, 0600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	if cfg.AppKey != "test-key-from-file" {
		t.Errorf("AppKey = %q, want %q", cfg.AppKey, "test-key-from-file")
	}
	if cfg.BaseURL != "https://custom.example.com" {
		t.Errorf("BaseURL = %q, want %q", cfg.BaseURL, "https://custom.example.com")
	}
}

// --- Test 54: Load env overrides file ---

func TestLoad_EnvOverridesFile(t *testing.T) {
	tmpDir := setHomeDir(t)

	configDir := filepath.Join(tmpDir, configDirName)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	content := []byte("app_key: file-key\nbase_url: https://file.example.com\n")
	if err := os.WriteFile(filepath.Join(configDir, configFileName), content, 0600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	t.Setenv("FLASHDUTY_APP_KEY", "env-key")
	t.Setenv("FLASHDUTY_BASE_URL", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	if cfg.AppKey != "env-key" {
		t.Errorf("AppKey = %q, want %q (env should override file)", cfg.AppKey, "env-key")
	}
}

// --- Test 55: Load FLASHDUTY_BASE_URL env ---

func TestLoad_BaseURLEnvOverridesFile(t *testing.T) {
	tmpDir := setHomeDir(t)

	configDir := filepath.Join(tmpDir, configDirName)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	content := []byte("base_url: https://file.example.com\n")
	if err := os.WriteFile(filepath.Join(configDir, configFileName), content, 0600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	t.Setenv("FLASHDUTY_APP_KEY", "")
	t.Setenv("FLASHDUTY_BASE_URL", "https://env.example.com")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	if cfg.BaseURL != "https://env.example.com" {
		t.Errorf("BaseURL = %q, want %q (env should override file)", cfg.BaseURL, "https://env.example.com")
	}
}

// --- Test 56: Load invalid YAML file ---

func TestLoad_InvalidYAML(t *testing.T) {
	tmpDir := setHomeDir(t)
	clearEnvVars(t)

	configDir := filepath.Join(tmpDir, configDirName)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	// Write invalid YAML content (tab indentation is invalid in YAML values context,
	// but a more reliable way to trigger a parse error is malformed structure).
	invalidYAML := []byte("app_key: [\ninvalid yaml\n")
	if err := os.WriteFile(filepath.Join(configDir, configFileName), invalidYAML, 0600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err := Load()
	if err == nil {
		t.Fatal("Load() expected error for invalid YAML, got nil")
	}
	if !strings.Contains(err.Error(), "failed to parse config file") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "failed to parse config file")
	}
}

// --- Test 57: Load empty BaseURL fallback ---

func TestLoad_EmptyBaseURLFallback(t *testing.T) {
	tmpDir := setHomeDir(t)
	clearEnvVars(t)

	configDir := filepath.Join(tmpDir, configDirName)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	// Explicitly set base_url to empty string in file.
	content := []byte("app_key: some-key\nbase_url: \"\"\n")
	if err := os.WriteFile(filepath.Join(configDir, configFileName), content, 0600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	if cfg.BaseURL != DefaultBaseURL {
		t.Errorf("BaseURL = %q, want %q (should fallback to default)", cfg.BaseURL, DefaultBaseURL)
	}
}

// --- Test 58: Save creates dir ---

func TestSave_CreatesDir(t *testing.T) {
	tmpDir := setHomeDir(t)
	clearEnvVars(t)

	cfg := &Config{
		AppKey:  "test-save-key",
		BaseURL: "https://save.example.com",
	}

	if err := Save(cfg); err != nil {
		t.Fatalf("Save() unexpected error: %v", err)
	}

	expectedDir := filepath.Join(tmpDir, configDirName)
	info, err := os.Stat(expectedDir)
	if err != nil {
		t.Fatalf("config directory was not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected config directory to be a directory")
	}
}

// --- Test 59: Save file permissions ---

func TestSave_FilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission checks are not reliable on Windows")
	}

	tmpDir := setHomeDir(t)
	clearEnvVars(t)

	cfg := &Config{
		AppKey:  "perm-test-key",
		BaseURL: DefaultBaseURL,
	}

	if err := Save(cfg); err != nil {
		t.Fatalf("Save() unexpected error: %v", err)
	}

	configFile := filepath.Join(tmpDir, configDirName, configFileName)
	info, err := os.Stat(configFile)
	if err != nil {
		t.Fatalf("config file not found: %v", err)
	}

	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("file permissions = %o, want 0600", perm)
	}
}

// --- Test 60: Save then Load roundtrip ---

func TestSave_LoadRoundtrip(t *testing.T) {
	setHomeDir(t)
	clearEnvVars(t)

	original := &Config{
		AppKey:  "roundtrip-key-12345",
		BaseURL: "https://roundtrip.example.com",
	}

	if err := Save(original); err != nil {
		t.Fatalf("Save() unexpected error: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	if loaded.AppKey != original.AppKey {
		t.Errorf("AppKey = %q, want %q", loaded.AppKey, original.AppKey)
	}
	if loaded.BaseURL != original.BaseURL {
		t.Errorf("BaseURL = %q, want %q", loaded.BaseURL, original.BaseURL)
	}
}

// --- Test 61: ConfigDir returns expected path ---

func TestConfigDir_ContainsFlashduty(t *testing.T) {
	setHomeDir(t)

	dir := ConfigDir()
	if !strings.Contains(dir, ".flashduty") {
		t.Errorf("ConfigDir() = %q, want it to contain %q", dir, ".flashduty")
	}
}

// --- Test 62: ConfigPath returns expected path ---

func TestConfigPath_ContainsConfigYAML(t *testing.T) {
	setHomeDir(t)

	path := ConfigPath()
	if !strings.Contains(path, "config.yaml") {
		t.Errorf("ConfigPath() = %q, want it to contain %q", path, "config.yaml")
	}
	if !strings.Contains(path, ".flashduty") {
		t.Errorf("ConfigPath() = %q, want it to contain %q", path, ".flashduty")
	}
}

// --- Test 63: ConfigSource app_key from env ---

func TestConfigSource_AppKeyFromEnv(t *testing.T) {
	setHomeDir(t)
	t.Setenv("FLASHDUTY_APP_KEY", "some-key")
	t.Setenv("FLASHDUTY_BASE_URL", "")

	got := ConfigSource("app_key")
	want := "(from env FLASHDUTY_APP_KEY)"
	if got != want {
		t.Errorf("ConfigSource(\"app_key\") = %q, want %q", got, want)
	}
}

// --- Test 64: ConfigSource base_url from env ---

func TestConfigSource_BaseURLFromEnv(t *testing.T) {
	setHomeDir(t)
	t.Setenv("FLASHDUTY_APP_KEY", "")
	t.Setenv("FLASHDUTY_BASE_URL", "https://env.example.com")

	got := ConfigSource("base_url")
	want := "(from env FLASHDUTY_BASE_URL)"
	if got != want {
		t.Errorf("ConfigSource(\"base_url\") = %q, want %q", got, want)
	}
}

// --- Test 65: ConfigSource from file ---

func TestConfigSource_FromFile(t *testing.T) {
	tmpDir := setHomeDir(t)
	clearEnvVars(t)

	configDir := filepath.Join(tmpDir, configDirName)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	configFile := filepath.Join(configDir, configFileName)
	if err := os.WriteFile(configFile, []byte("app_key: test\n"), 0600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	got := ConfigSource("app_key")
	wantPrefix := "(from "
	if !strings.HasPrefix(got, wantPrefix) {
		t.Errorf("ConfigSource(\"app_key\") = %q, want it to start with %q", got, wantPrefix)
	}
	if !strings.Contains(got, ConfigPath()) {
		t.Errorf("ConfigSource(\"app_key\") = %q, want it to contain %q", got, ConfigPath())
	}
}

// --- Test 66: ConfigSource default ---

func TestConfigSource_Default(t *testing.T) {
	setHomeDir(t)
	clearEnvVars(t)

	// No config file exists in the temp home directory.
	got := ConfigSource("app_key")
	want := "(default)"
	if got != want {
		t.Errorf("ConfigSource(\"app_key\") = %q, want %q", got, want)
	}
}

// --- Test 67: ConfigSource unknown key ---

func TestConfigSource_UnknownKey(t *testing.T) {
	setHomeDir(t)
	clearEnvVars(t)

	// Unknown key should not match any env var check, so it falls through
	// to file check or default.
	got := ConfigSource("unknown")

	// With no file present, it should return "(default)".
	want := "(default)"
	if got != want {
		t.Errorf("ConfigSource(\"unknown\") = %q, want %q", got, want)
	}
}

// TestConfigSource_UnknownKeyWithFile verifies that an unknown key still
// reports "(from <path>)" when the config file exists on disk, since the
// file-existence check is key-agnostic.
func TestConfigSource_UnknownKeyWithFile(t *testing.T) {
	tmpDir := setHomeDir(t)
	clearEnvVars(t)

	configDir := filepath.Join(tmpDir, configDirName)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, configFileName), []byte("app_key: x\n"), 0600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	got := ConfigSource("unknown")
	if !strings.HasPrefix(got, "(from ") {
		t.Errorf("ConfigSource(\"unknown\") with file present = %q, want prefix %q", got, "(from ")
	}
}
