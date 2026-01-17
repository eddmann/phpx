package executor

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/eddmann/phpx/internal/sandbox"
)

func TestToolRunner_runs_in_working_directory(t *testing.T) {
	// Create work directory (where tool should run)
	workDir, err := os.MkdirTemp("", "workdir")
	if err != nil {
		t.Fatalf("failed to create work dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(workDir) }()

	// Create tool directory (where tool is installed)
	toolDir, err := os.MkdirTemp("", "tooldir")
	if err != nil {
		t.Fatalf("failed to create tool dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(toolDir) }()

	// Create vendor/bin structure
	binDir := filepath.Join(toolDir, "vendor", "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("failed to create bin dir: %v", err)
	}

	// Create a script that writes marker in current directory
	scriptPath := filepath.Join(binDir, "marker")
	script := `#!/bin/sh
touch marker.txt
`
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	opts := &ToolOptions{
		PHPBinary:  "/bin/sh",
		ToolDir:    toolDir,
		BinaryName: "marker",
		Sandbox:    &sandbox.None{},
		Network:    true,
		WorkDir:    workDir,
		Timeout:    5 * time.Second,
	}

	runner := NewToolRunner(opts)
	result, err := runner.Run(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("exit code = %d, want 0", result.ExitCode)
	}

	// Marker should be in workDir, not toolDir
	markerInWorkDir := filepath.Join(workDir, "marker.txt")
	markerInToolDir := filepath.Join(toolDir, "marker.txt")

	if !fileExists(markerInWorkDir) {
		t.Error("marker.txt not created in working directory")
	}
	if fileExists(markerInToolDir) {
		t.Error("marker.txt incorrectly created in tool directory")
	}
}

func TestToolRunner_returns_exit_code(t *testing.T) {
	tests := []struct {
		name     string
		exitCode int
	}{
		{"exit code 1", 1},
		{"exit code 42", 42},
		{"exit code 127", 127},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toolDir, err := os.MkdirTemp("", "tooldir")
			if err != nil {
				t.Fatalf("failed to create tool dir: %v", err)
			}
			defer func() { _ = os.RemoveAll(toolDir) }()

			binDir := filepath.Join(toolDir, "vendor", "bin")
			if err := os.MkdirAll(binDir, 0755); err != nil {
				t.Fatalf("failed to create bin dir: %v", err)
			}

			scriptPath := filepath.Join(binDir, "failing")
			script := "#!/bin/sh\nexit " + strconv.Itoa(tt.exitCode) + "\n"
			if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
				t.Fatalf("failed to write script: %v", err)
			}

			opts := &ToolOptions{
				PHPBinary:  "/bin/sh",
				ToolDir:    toolDir,
				BinaryName: "failing",
				Sandbox:    &sandbox.None{},
				Network:    true,
				Timeout:    5 * time.Second,
			}

			runner := NewToolRunner(opts)
			result, err := runner.Run(context.Background())

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.ExitCode != tt.exitCode {
				t.Errorf("exit code = %d, want %d", result.ExitCode, tt.exitCode)
			}
		})
	}
}

func TestToolRunner_passes_arguments(t *testing.T) {
	toolDir, err := os.MkdirTemp("", "tooldir")
	if err != nil {
		t.Fatalf("failed to create tool dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(toolDir) }()

	binDir := filepath.Join(toolDir, "vendor", "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("failed to create bin dir: %v", err)
	}

	outputFile := filepath.Join(toolDir, "args.txt")
	scriptPath := filepath.Join(binDir, "argcheck")
	script := `#!/bin/sh
echo "$1 $2" > ` + outputFile + `
`
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	opts := &ToolOptions{
		PHPBinary:  "/bin/sh",
		ToolDir:    toolDir,
		BinaryName: "argcheck",
		Sandbox:    &sandbox.None{},
		Network:    true,
		Args:       []string{"analyze", "src/"},
		Timeout:    5 * time.Second,
	}

	runner := NewToolRunner(opts)
	result, err := runner.Run(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("exit code = %d, want 0", result.ExitCode)
	}

	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	if string(content) != "analyze src/\n" {
		t.Errorf("arguments = %q, want %q", string(content), "analyze src/\n")
	}
}

func TestToolRunner_constructs_binary_path(t *testing.T) {
	toolDir, err := os.MkdirTemp("", "tooldir")
	if err != nil {
		t.Fatalf("failed to create tool dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(toolDir) }()

	binDir := filepath.Join(toolDir, "vendor", "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("failed to create bin dir: %v", err)
	}

	// Create scripts with specific names to verify path construction
	for _, name := range []string{"phpstan", "psalm", "php-cs-fixer"} {
		scriptPath := filepath.Join(binDir, name)
		script := "#!/bin/sh\necho " + name + "\n"
		if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
			t.Fatalf("failed to write script: %v", err)
		}
	}

	tests := []struct {
		binaryName string
		wantOutput string
	}{
		{"phpstan", "phpstan\n"},
		{"psalm", "psalm\n"},
		{"php-cs-fixer", "php-cs-fixer\n"},
	}

	for _, tt := range tests {
		t.Run(tt.binaryName, func(t *testing.T) {
			opts := &ToolOptions{
				PHPBinary:  "/bin/sh",
				ToolDir:    toolDir,
				BinaryName: tt.binaryName,
				Sandbox:    &sandbox.None{},
				Network:    true,
				Timeout:    5 * time.Second,
			}

			runner := NewToolRunner(opts)
			result, err := runner.Run(context.Background())

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.ExitCode != 0 {
				t.Errorf("exit code = %d, want 0", result.ExitCode)
			}
			if result.Stdout != tt.wantOutput {
				t.Errorf("stdout = %q, want %q", result.Stdout, tt.wantOutput)
			}
		})
	}
}

func TestToolRunner_defaults_to_current_working_directory(t *testing.T) {
	toolDir, err := os.MkdirTemp("", "tooldir")
	if err != nil {
		t.Fatalf("failed to create tool dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(toolDir) }()

	binDir := filepath.Join(toolDir, "vendor", "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("failed to create bin dir: %v", err)
	}

	// Create a script that prints current working directory
	scriptPath := filepath.Join(binDir, "pwd")
	script := `#!/bin/sh
pwd
`
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	opts := &ToolOptions{
		PHPBinary:  "/bin/sh",
		ToolDir:    toolDir,
		BinaryName: "pwd",
		Sandbox:    &sandbox.None{},
		Network:    true,
		Timeout:    5 * time.Second,
		// WorkDir not set - should default to current directory
	}

	runner := NewToolRunner(opts)
	result, err := runner.Run(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("exit code = %d, want 0", result.ExitCode)
	}

	// Output should be the current working directory
	got := result.Stdout
	// Trim newline for comparison
	if len(got) > 0 && got[len(got)-1] == '\n' {
		got = got[:len(got)-1]
	}
	if got != cwd {
		t.Errorf("working directory = %q, want %q", got, cwd)
	}
}
