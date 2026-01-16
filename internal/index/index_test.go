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
			name: "returns exact version when specified",
			constraint: "8.3.10",
			want:       "8.3.10",
		},
		{
			name: "returns highest matching for gte constraint",
			constraint: ">=8.2",
			want:       "8.4.17",
		},
		{
			name: "returns highest in major for caret constraint",
			constraint: "^8.3",
			want:       "8.4.17",
		},
		{
			name: "returns highest patch for tilde constraint",
			constraint: "~8.3.0",
			want:       "8.3.17",
		},
		{
			name: "returns highest in range for compound constraint",
			constraint: ">=8.2, <8.4",
			want:       "8.3.17",
		},
		{
			name: "returns error when no version matches",
			constraint: ">=9.0",
			wantErr:    true,
		},
		{
			name: "returns error for invalid constraint syntax",
			constraint: "invalid",
			wantErr:    true,
		},
		{
			name: "handles single pipe or constraint",
			constraint: "^7.4|^8.0",
			want:       "8.4.17",
		},
		{
			name: "handles double pipe or constraint",
			constraint: "^7.4 || ^8.0",
			want:       "8.4.17",
		},
		{
			name: "handles multiple or branches",
			constraint: "^8.1|^8.2|^8.3|^8.4",
			want:       "8.4.17",
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
				t.Errorf("got %s, want %s", got.String(), tt.want)
			}
		})
	}
}

func TestLatestVersion(t *testing.T) {
	t.Run("returns highest version from list", func(t *testing.T) {
		versions := []*semver.Version{
			semver.MustParse("8.4.17"),
			semver.MustParse("8.3.17"),
			semver.MustParse("8.2.27"),
		}

		got := LatestVersion(versions)

		if got.String() != "8.4.17" {
			t.Errorf("got %s, want 8.4.17", got.String())
		}
	})

	t.Run("returns nil for empty list", func(t *testing.T) {
		got := LatestVersion(nil)

		if got != nil {
			t.Errorf("got %v, want nil", got)
		}
	})
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
			name: "returns common when no extensions requested",
			extensions: nil,
			want:       "common",
		},
		{
			name: "returns common when all extensions in common tier",
			extensions: []string{"redis", "curl"},
			want:       "common",
		},
		{
			name: "returns bulk when any extension requires bulk",
			extensions: []string{"redis", "imagick"},
			want:       "bulk",
		},
		{
			name: "returns error when extension unavailable",
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
				t.Errorf("got %s, want %s", got, tt.want)
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
			name: "returns latest composer for modern php",
			phpVersion: "8.4.17",
			want:       "2.9.3",
		},
		{
			name: "returns latest composer at exact minimum php",
			phpVersion: "7.2.5",
			want:       "2.9.3",
		},
		{
			name: "returns older composer for older php",
			phpVersion: "7.0.0",
			want:       "2.2.26",
		},
		{
			name: "returns error when php too old",
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
				t.Errorf("got %s, want %s", got.Version, tt.want)
			}
		})
	}
}

func TestOsName(t *testing.T) {
	t.Run("returns valid os name", func(t *testing.T) {
		name := osName()

		if name == "" {
			t.Error("got empty string, want valid os name")
		}
	})
}

func TestArchName(t *testing.T) {
	t.Run("returns valid arch name", func(t *testing.T) {
		name := archName()

		if name == "" {
			t.Error("got empty string, want valid arch name")
		}
	})
}
