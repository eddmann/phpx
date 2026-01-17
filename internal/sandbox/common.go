package sandbox

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/eddmann/phpx/internal/proxy"
	"github.com/eddmann/phpx/internal/util"
)

// ShellEscape escapes a string for safe use in shell commands.
// Wraps in single quotes and escapes embedded single quotes.
func ShellEscape(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// BuildPHPArgs constructs PHP command arguments from config.
func BuildPHPArgs(cfg *Config) []string {
	args := []string{cfg.PHPBinary}

	if cfg.MemoryMB > 0 {
		args = append(args, "-d", fmt.Sprintf("memory_limit=%dM", cfg.MemoryMB))
	}
	if cfg.CPUSeconds > 0 {
		args = append(args, "-d", fmt.Sprintf("max_execution_time=%d", cfg.CPUSeconds))
	}
	if cfg.AutoloadFile != "" {
		args = append(args, "-d", fmt.Sprintf("auto_prepend_file=%s", cfg.AutoloadFile))
	}

	args = append(args, cfg.ScriptPath)
	args = append(args, cfg.ScriptArgs...)

	return args
}

// BuildSocatBridgeCommand creates a shell command that starts socat to bridge
// localhost traffic to a Unix socket, then runs the given command.
// Uses a retry loop instead of sleep to avoid race conditions.
func BuildSocatBridgeCommand(socketPath string, phpCmd string) string {
	return fmt.Sprintf(
		`socat TCP-LISTEN:%d,fork,reuseaddr UNIX-CONNECT:%s &
SOCAT_PID=$!
for i in 1 2 3 4 5 6 7 8 9 10; do
  if nc -z 127.0.0.1 %d 2>/dev/null; then break; fi
  sleep 0.05
done
%s
EXIT_CODE=$?
kill $SOCAT_PID 2>/dev/null
exit $EXIT_CODE`,
		proxy.SandboxBridgePort,
		ShellEscape(socketPath),
		proxy.SandboxBridgePort,
		phpCmd,
	)
}

// BuildPHPCommand constructs an escaped PHP command string from config.
func BuildPHPCommand(cfg *Config) string {
	phpArgs := BuildPHPArgs(cfg)
	phpCmd := ShellEscape(cfg.PHPBinary)
	for _, arg := range phpArgs[1:] {
		phpCmd += " " + ShellEscape(arg)
	}
	return phpCmd
}

// ProxyEnvVars returns environment variables for the sandbox bridge proxy.
func ProxyEnvVars() []string {
	port := strconv.Itoa(proxy.SandboxBridgePort)
	return []string{
		"HTTP_PROXY=http://127.0.0.1:" + port,
		"HTTPS_PROXY=http://127.0.0.1:" + port,
		"http_proxy=http://127.0.0.1:" + port,
		"https_proxy=http://127.0.0.1:" + port,
		"ALL_PROXY=http://127.0.0.1:" + port,
	}
}

// ExecuteResult holds the execution result with buffered output.
type ExecuteResult struct {
	Result *Result
	Stdout *bytes.Buffer
	Stderr *bytes.Buffer
}

// SetupCommand configures a command with standard I/O handling and environment.
// Returns buffers for stdout/stderr if streaming is not configured.
func SetupCommand(cmd *exec.Cmd, cfg *Config) (*bytes.Buffer, *bytes.Buffer) {
	var stdout, stderr bytes.Buffer

	if cfg.Stdin != nil {
		cmd.Stdin = cfg.Stdin
	}
	if cfg.Stdout != nil {
		cmd.Stdout = cfg.Stdout
	} else {
		cmd.Stdout = &stdout
	}
	if cfg.Stderr != nil {
		cmd.Stderr = cfg.Stderr
	} else {
		cmd.Stderr = &stderr
	}

	// Build environment - use filtered safelist plus explicitly allowed vars
	env := util.FilterEnv(cfg.AllowedEnvVars)
	env = append(env, cfg.Env...)
	cmd.Env = env

	return &stdout, &stderr
}

// BuildResult creates a Result from command execution, extracting exit code and output.
func BuildResult(err error, cfg *Config, stdout, stderr *bytes.Buffer) (*Result, error) {
	result := &Result{}

	if cfg.Stdout == nil && stdout != nil {
		result.Stdout = stdout.String()
	}
	if cfg.Stderr == nil && stderr != nil {
		result.Stderr = stderr.String()
	}

	if exitErr, ok := err.(*exec.ExitError); ok {
		result.ExitCode = exitErr.ExitCode()
	} else if err != nil {
		return result, err
	}

	return result, nil
}
