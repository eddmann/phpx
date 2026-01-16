package cache

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDir(t *testing.T) {
	t.Run("returns path ending with phpx", func(t *testing.T) {
		dir, err := Dir()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.HasSuffix(dir, ".phpx") {
			t.Errorf("got %q, want suffix .phpx", dir)
		}
	})
}

func TestPHPPath(t *testing.T) {
	t.Run("returns path containing version and tier", func(t *testing.T) {
		path, err := PHPPath("8.4.17", "common")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(path, "8.4.17-common") {
			t.Errorf("got %q, want to contain 8.4.17-common", path)
		}
	})

	t.Run("returns path ending with bin php", func(t *testing.T) {
		path, err := PHPPath("8.4.17", "common")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.HasSuffix(path, filepath.Join("bin", "php")) {
			t.Errorf("got %q, want suffix bin/php", path)
		}
	})
}

func TestToolPath(t *testing.T) {
	t.Run("converts slashes to dashes in package name", func(t *testing.T) {
		path, err := ToolPath("phpstan/phpstan", "1.10.0")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if strings.Contains(path, "phpstan/phpstan") {
			t.Errorf("got %q, should not contain /", path)
		}
	})

	t.Run("includes package and version in path", func(t *testing.T) {
		path, err := ToolPath("phpstan/phpstan", "1.10.0")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(path, "phpstan-phpstan-1.10.0") {
			t.Errorf("got %q, want to contain phpstan-phpstan-1.10.0", path)
		}
	})
}

func TestDepsHash(t *testing.T) {
	tests := []struct {
		name     string
		packages []string
		wantSame bool
		compare  []string
	}{
		{
			name: "produces deterministic hash",
			packages: []string{"vendor/a:^1.0", "vendor/b:^2.0"},
			wantSame: true,
			compare:  []string{"vendor/a:^1.0", "vendor/b:^2.0"},
		},
		{
			name: "produces same hash regardless of order",
			packages: []string{"vendor/b:^2.0", "vendor/a:^1.0"},
			wantSame: true,
			compare:  []string{"vendor/a:^1.0", "vendor/b:^2.0"},
		},
		{
			name: "produces same hash regardless of case",
			packages: []string{"Vendor/A:^1.0"},
			wantSame: true,
			compare:  []string{"vendor/a:^1.0"},
		},
		{
			name: "produces different hash for different packages",
			packages: []string{"vendor/a:^1.0"},
			wantSame: false,
			compare:  []string{"vendor/b:^1.0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1 := DepsHash(tt.packages)
			hash2 := DepsHash(tt.compare)

			if tt.wantSame && hash1 != hash2 {
				t.Errorf("DepsHash(%v) = %s, DepsHash(%v) = %s, want same", tt.packages, hash1, tt.compare, hash2)
			}

			if !tt.wantSame && hash1 == hash2 {
				t.Errorf("DepsHash(%v) = DepsHash(%v), want different", tt.packages, tt.compare)
			}
		})
	}
}

func TestExists(t *testing.T) {
	t.Run("returns true for existing file", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "test")
		if err != nil {
			t.Fatalf("CreateTemp() error: %v", err)
		}
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		got := Exists(tmpFile.Name())

		if !got {
			t.Error("got false, want true")
		}
	})

	t.Run("returns false for nonexistent file", func(t *testing.T) {
		got := Exists("/nonexistent/path/file")

		if got {
			t.Error("got true, want false")
		}
	})
}

func TestEnsureDir(t *testing.T) {
	t.Run("creates nested directories", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test")
		if err != nil {
			t.Fatalf("MkdirTemp() error: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		newDir := filepath.Join(tmpDir, "a", "b", "c")

		err = EnsureDir(newDir)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !Exists(newDir) {
			t.Error("directory was not created")
		}
	})

	t.Run("succeeds when directory already exists", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test")
		if err != nil {
			t.Fatalf("MkdirTemp() error: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		err = EnsureDir(tmpDir)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}
