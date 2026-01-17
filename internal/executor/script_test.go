package executor

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/eddmann/phpx/internal/sandbox"
)

func TestScriptRunner_runs_in_script_directory(t *testing.T) {
	// Create a temp directory for the script
	scriptDir, err := os.MkdirTemp("", "scriptdir")
	if err != nil {
		t.Fatalf("failed to create script dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(scriptDir) }()

	// Create a script that writes a marker file in current directory
	scriptPath := filepath.Join(scriptDir, "script.sh")
	script := `#!/bin/sh
touch marker.txt
`
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	opts := &ScriptOptions{
		ScriptPath: scriptPath,
		PHPBinary:  "/bin/sh",
		Sandbox:    &sandbox.None{},
		Network:    true,
		Timeout:    5 * time.Second,
	}

	runner := NewScriptRunner(opts)
	result, err := runner.Run(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("exit code = %d, want 0", result.ExitCode)
	}

	// Marker should be in scriptDir (WorkDir is set to script's directory)
	markerPath := filepath.Join(scriptDir, "marker.txt")
	if !fileExists(markerPath) {
		t.Error("marker.txt not created in script directory")
	}
}

func TestScriptRunner_returns_exit_code(t *testing.T) {
	tests := []struct {
		name     string
		exitCode int
	}{
		{"exit code 1", 1},
		{"exit code 42", 42},
		{"exit code 99", 99},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "test")
			if err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}
			defer func() { _ = os.RemoveAll(tmpDir) }()

			scriptPath := filepath.Join(tmpDir, "exit.sh")
			script := "#!/bin/sh\nexit " + strconv.Itoa(tt.exitCode) + "\n"
			if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
				t.Fatalf("failed to write script: %v", err)
			}

			opts := &ScriptOptions{
				ScriptPath: scriptPath,
				PHPBinary:  "/bin/sh",
				Sandbox:    &sandbox.None{},
				Network:    true,
				Timeout:    5 * time.Second,
			}

			runner := NewScriptRunner(opts)
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

func TestScriptRunner_passes_arguments(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	outputFile := filepath.Join(tmpDir, "args.txt")
	scriptPath := filepath.Join(tmpDir, "args.sh")
	script := `#!/bin/sh
echo "$1 $2 $3" > ` + outputFile + `
`
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	opts := &ScriptOptions{
		ScriptPath: scriptPath,
		PHPBinary:  "/bin/sh",
		Sandbox:    &sandbox.None{},
		Network:    true,
		Args:       []string{"hello", "world", "test"},
		Timeout:    5 * time.Second,
	}

	runner := NewScriptRunner(opts)
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

	if string(content) != "hello world test\n" {
		t.Errorf("arguments = %q, want %q", string(content), "hello world test\n")
	}
}

func TestScriptRunner_captures_stdout(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	scriptPath := filepath.Join(tmpDir, "echo.sh")
	script := `#!/bin/sh
echo "hello stdout"
`
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	opts := &ScriptOptions{
		ScriptPath: scriptPath,
		PHPBinary:  "/bin/sh",
		Sandbox:    &sandbox.None{},
		Network:    true,
		Timeout:    5 * time.Second,
		// No Stdout writer - should capture to Result.Stdout
	}

	runner := NewScriptRunner(opts)
	result, err := runner.Run(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Stdout != "hello stdout\n" {
		t.Errorf("stdout = %q, want %q", result.Stdout, "hello stdout\n")
	}
}

func TestScriptRunner_streams_to_writers(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	scriptPath := filepath.Join(tmpDir, "echo.sh")
	script := `#!/bin/sh
echo "to stdout"
echo "to stderr" >&2
`
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	var stdout, stderr bytes.Buffer

	opts := &ScriptOptions{
		ScriptPath: scriptPath,
		PHPBinary:  "/bin/sh",
		Sandbox:    &sandbox.None{},
		Network:    true,
		Timeout:    5 * time.Second,
		Stdout:     &stdout,
		Stderr:     &stderr,
	}

	runner := NewScriptRunner(opts)
	result, err := runner.Run(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// When streaming, Result.Stdout/Stderr should be empty
	if result.Stdout != "" {
		t.Errorf("result.Stdout = %q, want empty", result.Stdout)
	}
	if result.Stderr != "" {
		t.Errorf("result.Stderr = %q, want empty", result.Stderr)
	}

	// Output should be in the buffers
	if stdout.String() != "to stdout\n" {
		t.Errorf("stdout buffer = %q, want %q", stdout.String(), "to stdout\n")
	}
	if stderr.String() != "to stderr\n" {
		t.Errorf("stderr buffer = %q, want %q", stderr.String(), "to stderr\n")
	}
}

func TestScriptRunner_uses_provided_timeout(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create a script that completes quickly
	scriptPath := filepath.Join(tmpDir, "quick.sh")
	script := `#!/bin/sh
echo "done"
`
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	opts := &ScriptOptions{
		ScriptPath: scriptPath,
		PHPBinary:  "/bin/sh",
		Sandbox:    &sandbox.None{},
		Network:    true,
		Timeout:    5 * time.Second,
	}

	runner := NewScriptRunner(opts)
	result, err := runner.Run(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("exit code = %d, want 0", result.ExitCode)
	}
	if result.Stdout != "done\n" {
		t.Errorf("stdout = %q, want %q", result.Stdout, "done\n")
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
