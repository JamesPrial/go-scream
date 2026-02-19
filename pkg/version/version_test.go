package version

import (
	"fmt"
	"testing"
)

// ---------------------------------------------------------------------------
// Default values
// ---------------------------------------------------------------------------

func TestDefaultVersion(t *testing.T) {
	if Version != "dev" {
		t.Errorf("Version = %q, want %q", Version, "dev")
	}
}

func TestDefaultCommit(t *testing.T) {
	if Commit != "unknown" {
		t.Errorf("Commit = %q, want %q", Commit, "unknown")
	}
}

func TestDefaultDate(t *testing.T) {
	if Date != "unknown" {
		t.Errorf("Date = %q, want %q", Date, "unknown")
	}
}

// ---------------------------------------------------------------------------
// String()
// ---------------------------------------------------------------------------

func TestString_DefaultValues(t *testing.T) {
	want := "dev (commit: unknown, built: unknown)"
	got := String()
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestString_Format(t *testing.T) {
	// Save original values and restore after test.
	origVersion := Version
	origCommit := Commit
	origDate := Date
	t.Cleanup(func() {
		Version = origVersion
		Commit = origCommit
		Date = origDate
	})

	tests := []struct {
		name    string
		version string
		commit  string
		date    string
		want    string
	}{
		{
			name:    "default values",
			version: "dev",
			commit:  "unknown",
			date:    "unknown",
			want:    "dev (commit: unknown, built: unknown)",
		},
		{
			name:    "release version",
			version: "1.0.0",
			commit:  "abc1234",
			date:    "2026-01-15T10:00:00Z",
			want:    "1.0.0 (commit: abc1234, built: 2026-01-15T10:00:00Z)",
		},
		{
			name:    "prerelease version",
			version: "0.5.0-beta.1",
			commit:  "deadbeef",
			date:    "2026-02-01",
			want:    "0.5.0-beta.1 (commit: deadbeef, built: 2026-02-01)",
		},
		{
			name:    "empty strings",
			version: "",
			commit:  "",
			date:    "",
			want:    " (commit: , built: )",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Version = tt.version
			Commit = tt.commit
			Date = tt.date

			got := String()
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestString_MatchesExpectedFormat(t *testing.T) {
	// Verify the format pattern: "VERSION (commit: COMMIT, built: DATE)"
	origVersion := Version
	origCommit := Commit
	origDate := Date
	t.Cleanup(func() {
		Version = origVersion
		Commit = origCommit
		Date = origDate
	})

	Version = "v2.3.4"
	Commit = "1234567"
	Date = "2026-03-01"

	expected := fmt.Sprintf("%s (commit: %s, built: %s)", Version, Commit, Date)
	got := String()
	if got != expected {
		t.Errorf("String() = %q, want %q", got, expected)
	}
}

// ---------------------------------------------------------------------------
// Benchmarks
// ---------------------------------------------------------------------------

func BenchmarkString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = String()
	}
}
