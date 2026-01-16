package exec

import (
	"os"
	"os/exec"
	"path/filepath"
)

// RunScript executes a PHP script with optional autoload prepending.
func RunScript(phpPath, scriptPath string, autoloadPath string, args []string) (int, error) {
	cmdArgs := []string{}

	// Add auto_prepend_file if we have dependencies
	if autoloadPath != "" {
		cmdArgs = append(cmdArgs, "-d", "auto_prepend_file="+autoloadPath)
	}

	cmdArgs = append(cmdArgs, scriptPath)
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.Command(phpPath, cmdArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	return exitCode(err), nil
}

// RunTool executes a tool binary from its installation directory.
func RunTool(phpPath, toolDir, binary string, args []string) (int, error) {
	binaryPath := filepath.Join(toolDir, "vendor", "bin", binary)

	cmdArgs := []string{binaryPath}
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.Command(phpPath, cmdArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	return exitCode(err), nil
}

// exitCode extracts the exit code from an exec error.
func exitCode(err error) int {
	if err == nil {
		return 0
	}

	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode()
	}

	// Default to 1 for other errors
	return 1
}
