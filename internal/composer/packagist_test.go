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
			name: "returns latest stable when no constraint",
			constraint: "",
			want:       "2.0.0",
		},
		{
			name: "returns exact version when specified",
			constraint: "1.10.0",
			want:       "1.10.0",
		},
		{
			name: "returns highest matching for caret constraint",
			constraint: "^1.9",
			want:       "1.10.5",
		},
		{
			name: "returns highest patch for tilde constraint",
			constraint: "~1.10.0",
			want:       "1.10.5",
		},
		{
			name: "returns error when no version matches",
			constraint: ">=3.0",
			wantErr:    true,
		},
		{
			name: "skips prerelease versions for stable constraint",
			constraint: "^1.0",
			want:       "1.10.5",
		},
		{
			name: "skips dev branches for stable constraint",
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
			name: "matches highest from multiple branches with single pipe",
			constraint: "^1.0|^2.0",
			want:       "2.1.0",
		},
		{
			name: "matches highest from multiple branches with double pipe",
			constraint: "^1.0 || ^2.0",
			want:       "2.1.0",
		},
		{
			name: "falls back to first branch when second has no match",
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
			name: "returns single binary basename",
			pkg:  "vendor/tool",
			bins: []string{"bin/tool"},
			want: "tool",
		},
		{
			name: "uses from flag when provided",
			pkg:      "vendor/tool",
			bins:     []string{"bin/other"},
			fromFlag: "custom",
			want:     "custom",
		},
		{
			name: "matches binary to package short name",
			pkg:  "phpstan/phpstan",
			bins: []string{"phpstan", "phpstan.phar"},
			want: "phpstan",
		},
		{
			name: "matches phar binary to package short name",
			pkg:  "phpstan/phpstan",
			bins: []string{"phpstan.phar", "other"},
			want: "phpstan.phar",
		},
		{
			name: "returns first binary when no name match",
			pkg:  "vendor/package",
			bins: []string{"first", "second"},
			want: "first",
		},
		{
			name: "returns error when no binaries declared",
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
			name: "expands phpstan alias",
			input: "phpstan",
			want:  "phpstan/phpstan",
		},
		{
			name: "expands psalm alias",
			input: "psalm",
			want:  "vimeo/psalm",
		},
		{
			name: "preserves full package name",
			input: "vendor/package",
			want:  "vendor/package",
		},
		{
			name: "preserves unknown short name",
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
			name: "parses package name only",
			arg:         "phpstan",
			wantPkg:     "phpstan",
			wantVersion: "",
		},
		{
			name: "parses package with at version",
			arg:         "phpstan@1.10.0",
			wantPkg:     "phpstan",
			wantVersion: "1.10.0",
		},
		{
			name: "parses package with colon constraint",
			arg:         "phpstan:^1.10",
			wantPkg:     "phpstan",
			wantVersion: "^1.10",
		},
		{
			name: "parses full package name with version",
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
			name: "converts single pipe to double pipe",
			input: "^7.4|^8.0",
			want:  "^7.4 || ^8.0",
		},
		{
			name: "preserves existing double pipes",
			input: "^7.4 || ^8.0",
			want:  "^7.4  ||  ^8.0",
		},
		{
			name: "returns constraint unchanged when no or",
			input: "^8.0",
			want:  "^8.0",
		},
		{
			name: "returns range constraint unchanged",
			input: ">=8.1",
			want:  ">=8.1",
		},
		{
			name: "handles complex or with and constraints",
			input: ">=7.2,<8.0|>=8.0,<9.0",
			want:  ">=7.2,<8.0 || >=8.0,<9.0",
		},
		{
			name: "handles multiple or branches",
			input: "^7.2|^7.3|^7.4|^8.0",
			want:  "^7.2 || ^7.3 || ^7.4 || ^8.0",
		},
		{
			name: "handles spaces around single pipe",
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
