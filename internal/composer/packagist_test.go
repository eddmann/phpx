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
			name:       "latest stable",
			constraint: "",
			want:       "2.0.0",
		},
		{
			name:       "exact version",
			constraint: "1.10.0",
			want:       "1.10.0",
		},
		{
			name:       "caret constraint",
			constraint: "^1.9",
			want:       "1.10.5",
		},
		{
			name:       "tilde constraint",
			constraint: "~1.10.0",
			want:       "1.10.5",
		},
		{
			name:       "no match",
			constraint: ">=3.0",
			wantErr:    true,
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
				t.Errorf("ResolveVersion() = %s, want %s", got.Version, tt.want)
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
			name: "single binary",
			pkg:  "vendor/tool",
			bins: []string{"bin/tool"},
			want: "tool",
		},
		{
			name:     "from flag",
			pkg:      "vendor/tool",
			bins:     []string{"bin/other"},
			fromFlag: "custom",
			want:     "custom",
		},
		{
			name: "match short name",
			pkg:  "phpstan/phpstan",
			bins: []string{"phpstan", "phpstan.phar"},
			want: "phpstan",
		},
		{
			name: "match short name with phar",
			pkg:  "phpstan/phpstan",
			bins: []string{"phpstan.phar", "other"},
			want: "phpstan.phar",
		},
		{
			name: "default to first",
			pkg:  "vendor/package",
			bins: []string{"first", "second"},
			want: "first",
		},
		{
			name:    "no binaries",
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
				t.Errorf("InferBinary() = %s, want %s", got, tt.want)
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
		{"phpstan alias", "phpstan", "phpstan/phpstan"},
		{"psalm alias", "psalm", "vimeo/psalm"},
		{"full name unchanged", "vendor/package", "vendor/package"},
		{"unknown unchanged", "unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ResolveAlias(tt.input); got != tt.want {
				t.Errorf("ResolveAlias(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseToolArg(t *testing.T) {
	tests := []struct {
		arg         string
		wantPkg     string
		wantVersion string
	}{
		{"phpstan", "phpstan", ""},
		{"phpstan@1.10.0", "phpstan", "1.10.0"},
		{"phpstan:^1.10", "phpstan", "^1.10"},
		{"vendor/package@2.0", "vendor/package", "2.0"},
	}

	for _, tt := range tests {
		t.Run(tt.arg, func(t *testing.T) {
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

func TestIsPrerelease(t *testing.T) {
	tests := []struct {
		version string
		want    bool
	}{
		{"1.0.0", false},
		{"1.0.0-alpha", true},
		{"1.0.0-beta", true},
		{"1.0.0-RC1", true},
		{"1.0.0-rc1", true},
		{"1.0.0-dev", true},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			if got := isPrerelease(tt.version); got != tt.want {
				t.Errorf("isPrerelease(%q) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}

func TestIsDev(t *testing.T) {
	tests := []struct {
		version string
		want    bool
	}{
		{"1.0.0", false},
		{"dev-main", true},
		{"dev-master", true},
		{"1.0.0-dev", false}, // This is a prerelease, not a dev branch
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			if got := isDev(tt.version); got != tt.want {
				t.Errorf("isDev(%q) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}
