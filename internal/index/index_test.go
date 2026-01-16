package index

import (
	"testing"

	"github.com/Masterminds/semver/v3"
)

func TestMatchingVersion(t *testing.T) {
	versions := []*semver.Version{
		semver.MustParse("8.4.17"),
		semver.MustParse("8.4.13"),
		semver.MustParse("8.3.17"),
		semver.MustParse("8.3.10"),
		semver.MustParse("8.2.27"),
		semver.MustParse("8.1.31"),
	}

	tests := []struct {
		name       string
		constraint string
		want       string
		wantErr    bool
	}{
		{
			name:       "exact version",
			constraint: "8.3.10",
			want:       "8.3.10",
		},
		{
			name:       "greater than or equal",
			constraint: ">=8.2",
			want:       "8.4.17",
		},
		{
			name:       "caret constraint",
			constraint: "^8.3",
			want:       "8.4.17", // ^8.3 means >=8.3.0, <9.0.0
		},
		{
			name:       "tilde constraint",
			constraint: "~8.3.0",
			want:       "8.3.17",
		},
		{
			name:       "range constraint",
			constraint: ">=8.2, <8.4",
			want:       "8.3.17",
		},
		{
			name:       "no match",
			constraint: ">=9.0",
			wantErr:    true,
		},
		{
			name:       "invalid constraint",
			constraint: "invalid",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MatchingVersion(versions, tt.constraint)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got.String() != tt.want {
				t.Errorf("MatchingVersion() = %s, want %s", got.String(), tt.want)
			}
		})
	}
}

func TestLatestVersion(t *testing.T) {
	versions := []*semver.Version{
		semver.MustParse("8.4.17"),
		semver.MustParse("8.3.17"),
		semver.MustParse("8.2.27"),
	}

	got := LatestVersion(versions)
	if got.String() != "8.4.17" {
		t.Errorf("LatestVersion() = %s, want 8.4.17", got.String())
	}

	if LatestVersion(nil) != nil {
		t.Error("LatestVersion(nil) should return nil")
	}
}

func TestRequiredTier(t *testing.T) {
	idx := &Index{
		CommonExtensions: []string{"redis", "curl", "pdo", "mbstring"},
		BulkExtensions:   []string{"redis", "curl", "pdo", "mbstring", "imagick", "intl", "swoole"},
	}

	tests := []struct {
		name       string
		extensions []string
		want       string
		wantErr    bool
	}{
		{
			name:       "no extensions",
			extensions: nil,
			want:       "common",
		},
		{
			name:       "common only",
			extensions: []string{"redis", "curl"},
			want:       "common",
		},
		{
			name:       "needs bulk",
			extensions: []string{"redis", "imagick"},
			want:       "bulk",
		},
		{
			name:       "unavailable extension",
			extensions: []string{"mongodb"},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := idx.RequiredTier(tt.extensions)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.want {
				t.Errorf("RequiredTier() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestSelectComposer(t *testing.T) {
	idx := &Index{
		ComposerVersions: []ComposerVersion{
			{Version: "2.9.3", MinPHP: 70205, Path: "/download/2.9.3/composer.phar"},
			{Version: "2.2.26", MinPHP: 50300, Path: "/download/2.2.26/composer.phar"},
		},
	}

	tests := []struct {
		name       string
		phpVersion string
		want       string
		wantErr    bool
	}{
		{
			name:       "PHP 8.4",
			phpVersion: "8.4.17",
			want:       "2.9.3",
		},
		{
			name:       "PHP 7.2.5 exact minimum",
			phpVersion: "7.2.5",
			want:       "2.9.3",
		},
		{
			name:       "PHP 7.0 falls back to 2.2",
			phpVersion: "7.0.0",
			want:       "2.2.26",
		},
		{
			name:       "PHP 5.2 too old",
			phpVersion: "5.2.0",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := idx.SelectComposer(tt.phpVersion)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got.Version != tt.want {
				t.Errorf("SelectComposer() = %s, want %s", got.Version, tt.want)
			}
		})
	}
}

func TestOsName(t *testing.T) {
	name := osName()
	// Should return either "macos" or "linux" on common platforms
	if name != "macos" && name != "linux" {
		// Just verify it returns something
		if name == "" {
			t.Error("osName() returned empty string")
		}
	}
}

func TestArchName(t *testing.T) {
	name := archName()
	// Should return x86_64 or aarch64 on common platforms
	if name != "x86_64" && name != "aarch64" {
		// Just verify it returns something
		if name == "" {
			t.Error("archName() returned empty string")
		}
	}
}
