package exec

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunTool_runs_in_callers_working_directory(t *testing.T) {
	// Create a temp directory to act as our "working directory"
	workDir, err := os.MkdirTemp("", "workdir")
	if err != nil {
		t.Fatalf("failed to create work dir: %v", err)
	}
	defer os.RemoveAll(workDir)

	// Create a separate temp directory to act as the "tool installation"
	toolDir, err := os.MkdirTemp("", "tooldir")
	if err != nil {
		t.Fatalf("failed to create tool dir: %v", err)
	}
	defer os.RemoveAll(toolDir)

	// Create vendor/bin directory structure
	binDir := filepath.Join(toolDir, "vendor", "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("failed to create bin dir: %v", err)
	}

	// Create a shell script that writes a marker file in the current directory
	scriptPath := filepath.Join(binDir, "marker")
	script := `#!/bin/sh
touch marker.txt
`
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	// Change to work directory before running
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("failed to change to work dir: %v", err)
	}
	defer os.Chdir(origDir)

	// Run the tool (using /bin/sh as the "PHP" binary)
	exitCode, err := RunTool("/bin/sh", toolDir, "marker", nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}

	// Verify marker file was created in workDir (not toolDir)
	markerInWorkDir := filepath.Join(workDir, "marker.txt")
	markerInToolDir := filepath.Join(toolDir, "marker.txt")

	if !fileExists(markerInWorkDir) {
		t.Error("marker.txt not created in working directory")
	}

	if fileExists(markerInToolDir) {
		t.Error("marker.txt incorrectly created in tool directory")
	}
}

func TestRunTool_returns_nonzero_exit_code(t *testing.T) {
	toolDir, err := os.MkdirTemp("", "tooldir")
	if err != nil {
		t.Fatalf("failed to create tool dir: %v", err)
	}
	defer os.RemoveAll(toolDir)

	binDir := filepath.Join(toolDir, "vendor", "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("failed to create bin dir: %v", err)
	}

	// Create a script that exits with code 42
	scriptPath := filepath.Join(binDir, "failing")
	script := `#!/bin/sh
exit 42
`
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	exitCode, err := RunTool("/bin/sh", toolDir, "failing", nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if exitCode != 42 {
		t.Errorf("exit code = %d, want 42", exitCode)
	}
}

func TestRunTool_passes_arguments_to_binary(t *testing.T) {
	toolDir, err := os.MkdirTemp("", "tooldir")
	if err != nil {
		t.Fatalf("failed to create tool dir: %v", err)
	}
	defer os.RemoveAll(toolDir)

	binDir := filepath.Join(toolDir, "vendor", "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("failed to create bin dir: %v", err)
	}

	// Create a script that writes its first argument to a file
	scriptPath := filepath.Join(binDir, "argcheck")
	script := `#!/bin/sh
echo "$1" > /tmp/phpx-test-arg.txt
`
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}
	defer os.Remove("/tmp/phpx-test-arg.txt")

	exitCode, err := RunTool("/bin/sh", toolDir, "argcheck", []string{"test-value"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}

	content, err := os.ReadFile("/tmp/phpx-test-arg.txt")
	if err != nil {
		t.Fatalf("failed to read arg file: %v", err)
	}

	if string(content) != "test-value\n" {
		t.Errorf("argument = %q, want %q", string(content), "test-value\n")
	}
}

func TestRunScript_runs_in_callers_working_directory(t *testing.T) {
	workDir, err := os.MkdirTemp("", "workdir")
	if err != nil {
		t.Fatalf("failed to create work dir: %v", err)
	}
	defer os.RemoveAll(workDir)

	// Create a script that writes a marker file
	scriptPath := filepath.Join(workDir, "script.sh")
	script := `#!/bin/sh
touch script-marker.txt
`
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	// Change to a different directory
	otherDir, err := os.MkdirTemp("", "otherdir")
	if err != nil {
		t.Fatalf("failed to create other dir: %v", err)
	}
	defer os.RemoveAll(otherDir)

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	if err := os.Chdir(otherDir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}
	defer os.Chdir(origDir)

	exitCode, err := RunScript("/bin/sh", scriptPath, "", nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}

	// Marker should be in otherDir (current working directory), not workDir (script location)
	markerInOtherDir := filepath.Join(otherDir, "script-marker.txt")
	markerInWorkDir := filepath.Join(workDir, "script-marker.txt")

	if !fileExists(markerInOtherDir) {
		t.Error("marker not created in current working directory")
	}

	if fileExists(markerInWorkDir) {
		t.Error("marker incorrectly created in script directory")
	}
}

func TestRunScript_returns_nonzero_exit_code(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	scriptPath := filepath.Join(tmpDir, "fail.sh")
	script := `#!/bin/sh
exit 99
`
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	exitCode, err := RunScript("/bin/sh", scriptPath, "", nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if exitCode != 99 {
		t.Errorf("exit code = %d, want 99", exitCode)
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
