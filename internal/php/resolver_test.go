package php

import (
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/phpx-dev/phpx/internal/index"
)

func TestResolve(t *testing.T) {
	idx := &index.Index{
		CommonVersions: []*semver.Version{
			semver.MustParse("8.4.17"),
			semver.MustParse("8.3.17"),
			semver.MustParse("8.2.27"),
		},
		BulkVersions: []*semver.Version{
			semver.MustParse("8.4.17"),
			semver.MustParse("8.3.17"),
			semver.MustParse("8.2.27"),
		},
		CommonExtensions: []string{"redis", "curl", "pdo"},
		BulkExtensions:   []string{"redis", "curl", "pdo", "imagick", "intl"},
	}

	tests := []struct {
		name        string
		constraint  string
		extensions  []string
		wantVersion string
		wantTier    string
		wantErr     bool
	}{
		{
			name: "returns latest common when no constraint or extensions",
			constraint:  "",
			extensions:  nil,
			wantVersion: "8.4.17",
			wantTier:    "common",
		},
		{
			name: "returns matching version for constraint",
			constraint:  "~8.3.0",
			extensions:  nil,
			wantVersion: "8.3.17",
			wantTier:    "common",
		},
		{
			name: "returns common tier for common extensions",
			constraint:  "",
			extensions:  []string{"redis", "curl"},
			wantVersion: "8.4.17",
			wantTier:    "common",
		},
		{
			name: "returns bulk tier when bulk extension needed",
			constraint:  "",
			extensions:  []string{"redis", "imagick"},
			wantVersion: "8.4.17",
			wantTier:    "bulk",
		},
		{
			name: "returns error for unavailable extension",
			constraint: "",
			extensions: []string{"mongodb"},
			wantErr:    true,
		},
		{
			name: "returns error when no version matches constraint",
			constraint: ">=9.0",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := Resolve(idx, tt.constraint, tt.extensions)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if res.Version.String() != tt.wantVersion {
				t.Errorf("Version = %s, want %s", res.Version.String(), tt.wantVersion)
			}

			if res.Tier != tt.wantTier {
				t.Errorf("Tier = %s, want %s", res.Tier, tt.wantTier)
			}
		})
	}
}
