package metadata

import (
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantPHP  string
		wantPkgs []string
		wantExts []string
		wantErr  bool
	}{
		{
			name: "full metadata block",
			content: `<?php
// phpx
// php = ">=8.2"
// packages = ["guzzlehttp/guzzle:^7.0", "monolog/monolog:^3.0"]
// extensions = ["redis", "gd"]

echo "Hello";
`,
			wantPHP:  ">=8.2",
			wantPkgs: []string{"guzzlehttp/guzzle:^7.0", "monolog/monolog:^3.0"},
			wantExts: []string{"redis", "gd"},
		},
		{
			name: "php version only",
			content: `<?php
// phpx
// php = "^8.3"

echo "Hello";
`,
			wantPHP:  "^8.3",
			wantPkgs: nil,
			wantExts: nil,
		},
		{
			name: "packages only",
			content: `<?php
// phpx
// packages = ["nesbot/carbon:^3.0"]

use Carbon\Carbon;
`,
			wantPHP:  "",
			wantPkgs: []string{"nesbot/carbon:^3.0"},
			wantExts: nil,
		},
		{
			name:     "no metadata block",
			content:  `<?php echo "Hello";`,
			wantPHP:  "",
			wantPkgs: nil,
			wantExts: nil,
		},
		{
			name: "empty phpx block",
			content: `<?php
// phpx

echo "Hello";
`,
			wantPHP:  "",
			wantPkgs: nil,
			wantExts: nil,
		},
		{
			name: "stops at non-comment line",
			content: `<?php
// phpx
// php = ">=8.2"
$x = 1;
// packages = ["should/ignore:^1.0"]
`,
			wantPHP:  ">=8.2",
			wantPkgs: nil,
			wantExts: nil,
		},
		{
			name: "invalid TOML",
			content: `<?php
// phpx
// php = invalid
`,
			wantErr: true,
		},
		{
			name: "whitespace variations",
			content: `<?php
// phpx
//php = ">=8.1"
//  packages = ["vendor/pkg:^1.0"]

echo "Hello";
`,
			wantPHP:  ">=8.1",
			wantPkgs: []string{"vendor/pkg:^1.0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta, err := Parse([]byte(tt.content))

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if meta.PHP != tt.wantPHP {
				t.Errorf("PHP = %q, want %q", meta.PHP, tt.wantPHP)
			}

			if !sliceEqual(meta.Packages, tt.wantPkgs) {
				t.Errorf("Packages = %v, want %v", meta.Packages, tt.wantPkgs)
			}

			if !sliceEqual(meta.Extensions, tt.wantExts) {
				t.Errorf("Extensions = %v, want %v", meta.Extensions, tt.wantExts)
			}
		})
	}
}

func sliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
