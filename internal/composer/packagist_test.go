package composer

import (
	"testing"
)

func TestResolveVersion(t *testing.T) {
	pkg := &PackageInfo{
		Name: "test/package",
		Versions: []PackageVersion{
			{Version: "2.0.0", Bin: []string{"bin/test"}},
			{Version: "1.10.5", Bin: []string{"bin/test"}},
			{Version: "1.10.0", Bin: []string{"bin/test"}},
			{Version: "1.9.0", Bin: []string{"bin/test"}},
			{Version: "1.0.0-alpha", Bin: []string{"bin/test"}},
			{Version: "dev-main", Bin: []string{"bin/test"}},
		},
	}

	tests := []struct {
		name       string
		constraint string
		want       string
		wantErr    bool
	}{
		{
			name:       "returns_latest_stable_when_no_constraint",
			constraint: "",
			want:       "2.0.0",
		},
		{
			name:       "returns_exact_version_when_specified",
			constraint: "1.10.0",
			want:       "1.10.0",
		},
		{
			name:       "returns_highest_matching_for_caret_constraint",
			constraint: "^1.9",
			want:       "1.10.5",
		},
		{
			name:       "returns_highest_patch_for_tilde_constraint",
			constraint: "~1.10.0",
			want:       "1.10.5",
		},
		{
			name:       "returns_error_when_no_version_matches",
			constraint: ">=3.0",
			wantErr:    true,
		},
		{
			name:       "skips_prerelease_versions_for_stable_constraint",
			constraint: "^1.0",
			want:       "1.10.5",
		},
		{
			name:       "skips_dev_branches_for_stable_constraint",
			constraint: "^2.0",
			want:       "2.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveVersion(pkg, tt.constraint)

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

func TestResolveVersion_with_or_constraints(t *testing.T) {
	pkg := &PackageInfo{
		Name: "test/package",
		Versions: []PackageVersion{
			{Version: "2.1.0", Bin: []string{"bin/test"}},
			{Version: "2.0.0", Bin: []string{"bin/test"}},
			{Version: "1.5.0", Bin: []string{"bin/test"}},
			{Version: "1.0.0", Bin: []string{"bin/test"}},
		},
	}

	tests := []struct {
		name       string
		constraint string
		want       string
		wantErr    bool
	}{
		{
			name:       "matches_highest_from_multiple_branches_with_single_pipe",
			constraint: "^1.0|^2.0",
			want:       "2.1.0",
		},
		{
			name:       "matches_highest_from_multiple_branches_with_double_pipe",
			constraint: "^1.0 || ^2.0",
			want:       "2.1.0",
		},
		{
			name:       "falls_back_to_first_branch_when_second_has_no_match",
			constraint: "^1.0|^3.0",
			want:       "1.5.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveVersion(pkg, tt.constraint)

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

func TestInferBinary(t *testing.T) {
	tests := []struct {
		name     string
		pkg      string
		bins     []string
		fromFlag string
		want     string
		wantErr  bool
	}{
		{
			name: "returns_single_binary_basename",
			pkg:  "vendor/tool",
			bins: []string{"bin/tool"},
			want: "tool",
		},
		{
			name:     "uses_from_flag_when_provided",
			pkg:      "vendor/tool",
			bins:     []string{"bin/other"},
			fromFlag: "custom",
			want:     "custom",
		},
		{
			name: "matches_binary_to_package_short_name",
			pkg:  "phpstan/phpstan",
			bins: []string{"phpstan", "phpstan.phar"},
			want: "phpstan",
		},
		{
			name: "matches_phar_binary_to_package_short_name",
			pkg:  "phpstan/phpstan",
			bins: []string{"phpstan.phar", "other"},
			want: "phpstan.phar",
		},
		{
			name: "returns_first_binary_when_no_name_match",
			pkg:  "vendor/package",
			bins: []string{"first", "second"},
			want: "first",
		},
		{
			name:    "returns_error_when_no_binaries_declared",
			pkg:     "vendor/library",
			bins:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := InferBinary(tt.pkg, tt.bins, tt.fromFlag)

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

func TestResolveAlias(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "expands_phpstan_alias",
			input: "phpstan",
			want:  "phpstan/phpstan",
		},
		{
			name:  "expands_psalm_alias",
			input: "psalm",
			want:  "vimeo/psalm",
		},
		{
			name:  "preserves_full_package_name",
			input: "vendor/package",
			want:  "vendor/package",
		},
		{
			name:  "preserves_unknown_short_name",
			input: "unknown",
			want:  "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveAlias(tt.input)

			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseToolArg(t *testing.T) {
	tests := []struct {
		name        string
		arg         string
		wantPkg     string
		wantVersion string
	}{
		{
			name:        "parses_package_name_only",
			arg:         "phpstan",
			wantPkg:     "phpstan",
			wantVersion: "",
		},
		{
			name:        "parses_package_with_at_version",
			arg:         "phpstan@1.10.0",
			wantPkg:     "phpstan",
			wantVersion: "1.10.0",
		},
		{
			name:        "parses_package_with_colon_constraint",
			arg:         "phpstan:^1.10",
			wantPkg:     "phpstan",
			wantVersion: "^1.10",
		},
		{
			name:        "parses_full_package_name_with_version",
			arg:         "vendor/package@2.0",
			wantPkg:     "vendor/package",
			wantVersion: "2.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg, version := ParseToolArg(tt.arg)

			if pkg != tt.wantPkg {
				t.Errorf("pkg = %q, want %q", pkg, tt.wantPkg)
			}

			if version != tt.wantVersion {
				t.Errorf("version = %q, want %q", version, tt.wantVersion)
			}
		})
	}
}

func TestNormalizeConstraint(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "converts_single_pipe_to_double_pipe",
			input: "^7.4|^8.0",
			want:  "^7.4 || ^8.0",
		},
		{
			name:  "preserves_existing_double_pipes",
			input: "^7.4 || ^8.0",
			want:  "^7.4  ||  ^8.0",
		},
		{
			name:  "returns_constraint_unchanged_when_no_or",
			input: "^8.0",
			want:  "^8.0",
		},
		{
			name:  "returns_range_constraint_unchanged",
			input: ">=8.1",
			want:  ">=8.1",
		},
		{
			name:  "handles_complex_or_with_and_constraints",
			input: ">=7.2,<8.0|>=8.0,<9.0",
			want:  ">=7.2,<8.0 || >=8.0,<9.0",
		},
		{
			name:  "handles_multiple_or_branches",
			input: "^7.2|^7.3|^7.4|^8.0",
			want:  "^7.2 || ^7.3 || ^7.4 || ^8.0",
		},
		{
			name:  "handles_spaces_around_single_pipe",
			input: "^7.4 | ^8.0",
			want:  "^7.4  ||  ^8.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeConstraint(tt.input)

			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
