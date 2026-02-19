package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// ---------------------------------------------------------------------------
// parseOpenClawConfig()
// ---------------------------------------------------------------------------

func Test_parseOpenClawConfig_Cases(t *testing.T) {
	tests := []struct {
		name      string
		content   string // file content to write; empty means no file created
		noFile    bool   // if true, use a nonexistent path
		wantToken string
		wantErr   bool
	}{
		{
			name:      "valid JSON with token",
			content:   `{"channels":{"discord":{"token":"my-token"}}}`,
			wantToken: "my-token",
			wantErr:   false,
		},
		{
			name:    "missing file returns error",
			noFile:  true,
			wantErr: true,
		},
		{
			name:    "invalid JSON returns error",
			content: `{invalid`,
			wantErr: true,
		},
		{
			name:      "missing token field returns empty string",
			content:   `{"channels":{"discord":{}}}`,
			wantToken: "",
			wantErr:   false,
		},
		{
			name:      "empty channels returns empty string",
			content:   `{"channels":{}}`,
			wantToken: "",
			wantErr:   false,
		},
		{
			name:      "empty object returns empty string",
			content:   `{}`,
			wantToken: "",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var path string
			if tt.noFile {
				// Use a path that definitely does not exist.
				path = filepath.Join(t.TempDir(), "nonexistent", "openclaw.json")
			} else {
				dir := t.TempDir()
				path = filepath.Join(dir, "openclaw.json")
				if err := os.WriteFile(path, []byte(tt.content), 0644); err != nil {
					t.Fatalf("failed to write test file: %v", err)
				}
			}

			got, err := parseOpenClawConfig(path)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseOpenClawConfig() expected error, got nil")
				}
				// When an error is expected, the token should be empty.
				if got != "" {
					t.Errorf("parseOpenClawConfig() token = %q, want empty on error", got)
				}
				return
			}

			if err != nil {
				t.Fatalf("parseOpenClawConfig() unexpected error: %v", err)
			}
			if got != tt.wantToken {
				t.Errorf("parseOpenClawConfig() = %q, want %q", got, tt.wantToken)
			}
		})
	}
}

func Test_parseOpenClawConfig_ValidJSON_StructureVerification(t *testing.T) {
	// Verify the JSON structure matches the expected schema by round-tripping.
	type discordChannel struct {
		Token string `json:"token"`
	}
	type channels struct {
		Discord discordChannel `json:"discord"`
	}
	type openClawConfig struct {
		Channels channels `json:"channels"`
	}

	cfg := openClawConfig{
		Channels: channels{
			Discord: discordChannel{
				Token: "round-trip-token",
			},
		},
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("json.Marshal() unexpected error: %v", err)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "openclaw.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	got, err := parseOpenClawConfig(path)
	if err != nil {
		t.Fatalf("parseOpenClawConfig() unexpected error: %v", err)
	}
	if got != "round-trip-token" {
		t.Errorf("parseOpenClawConfig() = %q, want %q", got, "round-trip-token")
	}
}

func Test_parseOpenClawConfig_ExtraFieldsIgnored(t *testing.T) {
	// Config may contain additional fields the skill wrapper does not use.
	content := `{
		"channels": {
			"discord": {
				"token": "extra-fields-token",
				"guild_id": "123456"
			},
			"slack": {
				"token": "slack-token"
			}
		},
		"version": "1.0"
	}`

	dir := t.TempDir()
	path := filepath.Join(dir, "openclaw.json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	got, err := parseOpenClawConfig(path)
	if err != nil {
		t.Fatalf("parseOpenClawConfig() unexpected error: %v", err)
	}
	if got != "extra-fields-token" {
		t.Errorf("parseOpenClawConfig() = %q, want %q", got, "extra-fields-token")
	}
}

func Test_parseOpenClawConfig_EmptyToken(t *testing.T) {
	// Token field is present but set to empty string.
	content := `{"channels":{"discord":{"token":""}}}`

	dir := t.TempDir()
	path := filepath.Join(dir, "openclaw.json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	got, err := parseOpenClawConfig(path)
	if err != nil {
		t.Fatalf("parseOpenClawConfig() unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("parseOpenClawConfig() = %q, want empty string", got)
	}
}

func Test_parseOpenClawConfig_EmptyFile(t *testing.T) {
	// An empty file is not valid JSON.
	dir := t.TempDir()
	path := filepath.Join(dir, "openclaw.json")
	if err := os.WriteFile(path, []byte(""), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err := parseOpenClawConfig(path)
	if err == nil {
		t.Fatal("parseOpenClawConfig() expected error for empty file, got nil")
	}
}

func Test_parseOpenClawConfig_NullValues(t *testing.T) {
	// JSON with explicit null values.
	tests := []struct {
		name      string
		content   string
		wantToken string
	}{
		{
			name:      "null channels",
			content:   `{"channels":null}`,
			wantToken: "",
		},
		{
			name:      "null discord",
			content:   `{"channels":{"discord":null}}`,
			wantToken: "",
		},
		{
			name:      "null token",
			content:   `{"channels":{"discord":{"token":null}}}`,
			wantToken: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "openclaw.json")
			if err := os.WriteFile(path, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			got, err := parseOpenClawConfig(path)
			if err != nil {
				t.Fatalf("parseOpenClawConfig() unexpected error: %v", err)
			}
			if got != tt.wantToken {
				t.Errorf("parseOpenClawConfig() = %q, want %q", got, tt.wantToken)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// resolveToken()
// ---------------------------------------------------------------------------

func Test_resolveToken_Cases(t *testing.T) {
	tests := []struct {
		name       string
		envToken   string // value for DISCORD_TOKEN; empty means unset
		setEnv     bool   // whether to set the env var at all
		fileToken  string // token to write in JSON file; empty means no file
		createFile bool   // whether to create the config file
		want       string
	}{
		{
			name:       "env var takes priority over file",
			envToken:   "env-token",
			setEnv:     true,
			fileToken:  "file-token",
			createFile: true,
			want:       "env-token",
		},
		{
			name:       "env empty falls back to file",
			envToken:   "",
			setEnv:     true,
			fileToken:  "file-token",
			createFile: true,
			want:       "file-token",
		},
		{
			name:       "env empty and no file returns empty",
			envToken:   "",
			setEnv:     true,
			fileToken:  "",
			createFile: false,
			want:       "",
		},
		{
			name:       "env set and no file returns env token",
			envToken:   "env-token",
			setEnv:     true,
			fileToken:  "",
			createFile: false,
			want:       "env-token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set or unset the DISCORD_TOKEN environment variable.
			if tt.setEnv {
				t.Setenv("DISCORD_TOKEN", tt.envToken)
			}

			var configPath string
			if tt.createFile {
				dir := t.TempDir()
				configPath = filepath.Join(dir, "openclaw.json")
				content := `{"channels":{"discord":{"token":"` + tt.fileToken + `"}}}`
				if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
					t.Fatalf("failed to write test config: %v", err)
				}
			} else {
				// Point at a nonexistent file.
				configPath = filepath.Join(t.TempDir(), "nonexistent", "openclaw.json")
			}

			got := resolveToken(configPath)
			if got != tt.want {
				t.Errorf("resolveToken() = %q, want %q", got, tt.want)
			}
		})
	}
}

func Test_resolveToken_EnvPriority(t *testing.T) {
	// Explicitly verify that DISCORD_TOKEN environment variable takes priority
	// even when both the env var and config file contain valid tokens.
	t.Setenv("DISCORD_TOKEN", "env-wins")

	dir := t.TempDir()
	configPath := filepath.Join(dir, "openclaw.json")
	content := `{"channels":{"discord":{"token":"file-loses"}}}`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	got := resolveToken(configPath)
	if got != "env-wins" {
		t.Errorf("resolveToken() = %q, want %q (env should take priority)", got, "env-wins")
	}
}

func Test_resolveToken_FallbackToFile(t *testing.T) {
	// When DISCORD_TOKEN is not set, the file token should be returned.
	t.Setenv("DISCORD_TOKEN", "")

	dir := t.TempDir()
	configPath := filepath.Join(dir, "openclaw.json")
	content := `{"channels":{"discord":{"token":"fallback-token"}}}`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	got := resolveToken(configPath)
	if got != "fallback-token" {
		t.Errorf("resolveToken() = %q, want %q", got, "fallback-token")
	}
}

func Test_resolveToken_NoSources(t *testing.T) {
	// Neither env var nor file provides a token.
	t.Setenv("DISCORD_TOKEN", "")

	configPath := filepath.Join(t.TempDir(), "nonexistent", "openclaw.json")

	got := resolveToken(configPath)
	if got != "" {
		t.Errorf("resolveToken() = %q, want empty string", got)
	}
}

func Test_resolveToken_InvalidJSON_FallsGracefully(t *testing.T) {
	// If the config file contains invalid JSON, resolveToken should not panic
	// and should return empty string (since the env var is also empty).
	t.Setenv("DISCORD_TOKEN", "")

	dir := t.TempDir()
	configPath := filepath.Join(dir, "openclaw.json")
	if err := os.WriteFile(configPath, []byte(`{broken json`), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	got := resolveToken(configPath)
	if got != "" {
		t.Errorf("resolveToken() = %q, want empty string when config is invalid", got)
	}
}

func Test_resolveToken_EnvOverridesInvalidFile(t *testing.T) {
	// Even when the config file is broken, the env var should still work.
	t.Setenv("DISCORD_TOKEN", "env-still-works")

	dir := t.TempDir()
	configPath := filepath.Join(dir, "openclaw.json")
	if err := os.WriteFile(configPath, []byte(`{broken json`), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	got := resolveToken(configPath)
	if got != "env-still-works" {
		t.Errorf("resolveToken() = %q, want %q", got, "env-still-works")
	}
}

func Test_resolveToken_FileWithEmptyToken(t *testing.T) {
	// File exists and is valid JSON, but the token field is empty.
	t.Setenv("DISCORD_TOKEN", "")

	dir := t.TempDir()
	configPath := filepath.Join(dir, "openclaw.json")
	content := `{"channels":{"discord":{"token":""}}}`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	got := resolveToken(configPath)
	if got != "" {
		t.Errorf("resolveToken() = %q, want empty string", got)
	}
}
