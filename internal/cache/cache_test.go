package cache

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDir(t *testing.T) {
	dir, err := Dir()
	if err != nil {
		t.Fatalf("Dir() error: %v", err)
	}

	if !strings.HasSuffix(dir, ".phpx") {
		t.Errorf("Dir() = %q, want suffix .phpx", dir)
	}
}

func TestPHPPath(t *testing.T) {
	path, err := PHPPath("8.4.17", "common")
	if err != nil {
		t.Fatalf("PHPPath() error: %v", err)
	}

	if !strings.Contains(path, "8.4.17-common") {
		t.Errorf("PHPPath() = %q, want to contain 8.4.17-common", path)
	}

	if !strings.HasSuffix(path, filepath.Join("bin", "php")) {
		t.Errorf("PHPPath() = %q, want suffix bin/php", path)
	}
}

func TestToolPath(t *testing.T) {
	path, err := ToolPath("phpstan/phpstan", "1.10.0")
	if err != nil {
		t.Fatalf("ToolPath() error: %v", err)
	}

	// Should convert / to -
	if strings.Contains(path, "phpstan/phpstan") {
		t.Errorf("ToolPath() = %q, should not contain /", path)
	}

	if !strings.Contains(path, "phpstan-phpstan-1.10.0") {
		t.Errorf("ToolPath() = %q, want to contain phpstan-phpstan-1.10.0", path)
	}
}

func TestDepsHash(t *testing.T) {
	tests := []struct {
		name     string
		packages []string
		wantSame bool
		compare  []string
	}{
		{
			name:     "deterministic",
			packages: []string{"vendor/a:^1.0", "vendor/b:^2.0"},
			wantSame: true,
			compare:  []string{"vendor/a:^1.0", "vendor/b:^2.0"},
		},
		{
			name:     "order independent",
			packages: []string{"vendor/b:^2.0", "vendor/a:^1.0"},
			wantSame: true,
			compare:  []string{"vendor/a:^1.0", "vendor/b:^2.0"},
		},
		{
			name:     "case insensitive",
			packages: []string{"Vendor/A:^1.0"},
			wantSame: true,
			compare:  []string{"vendor/a:^1.0"},
		},
		{
			name:     "different packages",
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
	// Test existing file
	tmpFile, err := os.CreateTemp("", "test")
	if err != nil {
		t.Fatalf("CreateTemp() error: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	if !Exists(tmpFile.Name()) {
		t.Error("Exists() = false for existing file")
	}

	// Test non-existing file
	if Exists("/nonexistent/path/file") {
		t.Error("Exists() = true for non-existing file")
	}
}

func TestEnsureDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatalf("MkdirTemp() error: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	newDir := filepath.Join(tmpDir, "a", "b", "c")
	if err := EnsureDir(newDir); err != nil {
		t.Fatalf("EnsureDir() error: %v", err)
	}

	if !Exists(newDir) {
		t.Error("EnsureDir() did not create directory")
	}

	// Should not error on existing dir
	if err := EnsureDir(newDir); err != nil {
		t.Errorf("EnsureDir() on existing dir error: %v", err)
	}
}
