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
			name:       "returns_exact_version_when_specified",
			constraint: "8.3.10",
			want:       "8.3.10",
		},
		{
			name:       "returns_highest_matching_for_gte_constraint",
			constraint: ">=8.2",
			want:       "8.4.17",
		},
		{
			name:       "returns_highest_in_major_for_caret_constraint",
			constraint: "^8.3",
			want:       "8.4.17",
		},
		{
			name:       "returns_highest_patch_for_tilde_constraint",
			constraint: "~8.3.0",
			want:       "8.3.17",
		},
		{
			name:       "returns_highest_in_range_for_compound_constraint",
			constraint: ">=8.2, <8.4",
			want:       "8.3.17",
		},
		{
			name:       "returns_error_when_no_version_matches",
			constraint: ">=9.0",
			wantErr:    true,
		},
		{
			name:       "returns_error_for_invalid_constraint_syntax",
			constraint: "invalid",
			wantErr:    true,
		},
		{
			name:       "handles_single_pipe_or_constraint",
			constraint: "^7.4|^8.0",
			want:       "8.4.17",
		},
		{
			name:       "handles_double_pipe_or_constraint",
			constraint: "^7.4 || ^8.0",
			want:       "8.4.17",
		},
		{
			name:       "handles_multiple_or_branches",
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
	t.Run("returns_highest_version_from_list", func(t *testing.T) {
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

	t.Run("returns_nil_for_empty_list", func(t *testing.T) {
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
			name:       "returns_common_when_no_extensions_requested",
			extensions: nil,
			want:       "common",
		},
		{
			name:       "returns_common_when_all_extensions_in_common_tier",
			extensions: []string{"redis", "curl"},
			want:       "common",
		},
		{
			name:       "returns_bulk_when_any_extension_requires_bulk",
			extensions: []string{"redis", "imagick"},
			want:       "bulk",
		},
		{
			name:       "returns_error_when_extension_unavailable",
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
			name:       "returns_latest_composer_for_modern_php",
			phpVersion: "8.4.17",
			want:       "2.9.3",
		},
		{
			name:       "returns_latest_composer_at_exact_minimum_php",
			phpVersion: "7.2.5",
			want:       "2.9.3",
		},
		{
			name:       "returns_older_composer_for_older_php",
			phpVersion: "7.0.0",
			want:       "2.2.26",
		},
		{
			name:       "returns_error_when_php_too_old",
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
	t.Run("returns_valid_os_name", func(t *testing.T) {
		name := osName()

		if name == "" {
			t.Error("got empty string, want valid os name")
		}
	})
}

func TestArchName(t *testing.T) {
	t.Run("returns_valid_arch_name", func(t *testing.T) {
		name := archName()

		if name == "" {
			t.Error("got empty string, want valid arch name")
		}
	})
}
