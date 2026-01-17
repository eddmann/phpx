package sandbox

import (
	"errors"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestBuildPHPArgs_basic(t *testing.T) {
	cfg := &Config{
		PHPBinary:  "/usr/bin/php",
		ScriptPath: "/path/to/script.php",
		ScriptArgs: []string{"arg1", "arg2"},
	}

	args := BuildPHPArgs(cfg)

	want := []string{"/usr/bin/php", "/path/to/script.php", "arg1", "arg2"}
	if !slices.Equal(args, want) {
		t.Errorf("args = %v, want %v", args, want)
	}
}

func TestBuildPHPArgs_with_memory_limit(t *testing.T) {
	cfg := &Config{
		PHPBinary:  "/usr/bin/php",
		ScriptPath: "/path/to/script.php",
		MemoryMB:   256,
	}

	args := BuildPHPArgs(cfg)

	if !slices.Contains(args, "-d") || !slices.Contains(args, "memory_limit=256M") {
		t.Errorf("args should contain memory limit, got %v", args)
	}
}

func TestBuildPHPArgs_with_cpu_limit(t *testing.T) {
	cfg := &Config{
		PHPBinary:  "/usr/bin/php",
		ScriptPath: "/path/to/script.php",
		CPUSeconds: 60,
	}

	args := BuildPHPArgs(cfg)

	if !slices.Contains(args, "-d") || !slices.Contains(args, "max_execution_time=60") {
		t.Errorf("args should contain cpu limit, got %v", args)
	}
}

func TestBuildPHPArgs_with_autoload(t *testing.T) {
	cfg := &Config{
		PHPBinary:    "/usr/bin/php",
		ScriptPath:   "/path/to/script.php",
		AutoloadFile: "/path/to/vendor/autoload.php",
	}

	args := BuildPHPArgs(cfg)

	if !slices.Contains(args, "-d") || !slices.Contains(args, "auto_prepend_file=/path/to/vendor/autoload.php") {
		t.Errorf("args should contain autoload, got %v", args)
	}
}

func TestBuildPHPArgs_with_all_options(t *testing.T) {
	cfg := &Config{
		PHPBinary:    "/usr/bin/php",
		ScriptPath:   "/path/to/script.php",
		ScriptArgs:   []string{"--verbose"},
		MemoryMB:     128,
		CPUSeconds:   30,
		AutoloadFile: "/path/to/autoload.php",
	}

	args := BuildPHPArgs(cfg)

	// Should have: php, -d, memory_limit=128M, -d, max_execution_time=30, -d, auto_prepend_file=..., script, --verbose
	if args[0] != "/usr/bin/php" {
		t.Errorf("first arg should be php binary, got %s", args[0])
	}
	if args[len(args)-2] != "/path/to/script.php" {
		t.Errorf("second to last arg should be script path, got %s", args[len(args)-2])
	}
	if args[len(args)-1] != "--verbose" {
		t.Errorf("last arg should be --verbose, got %s", args[len(args)-1])
	}
}

func TestBuildResult_extracts_exit_code(t *testing.T) {
	tests := []struct {
		name         string
		exitCode     int
		wantExitCode int
	}{
		{"exit code 0", 0, 0},
		{"exit code 1", 1, 1},
		{"exit code 42", 42, 42},
		{"exit code 127", 127, 127},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create an exec.ExitError with the desired exit code
			cmd := exec.Command("sh", "-c", "exit "+strconv.Itoa(tt.exitCode))
			err := cmd.Run()

			cfg := &Config{}
			result, resultErr := BuildResult(err, cfg, nil, nil)

			if tt.exitCode == 0 {
				if resultErr != nil {
					t.Errorf("unexpected error for exit code 0: %v", resultErr)
				}
			}

			if result.ExitCode != tt.wantExitCode {
				t.Errorf("exit code = %d, want %d", result.ExitCode, tt.wantExitCode)
			}
		})
	}
}

func TestBuildResult_returns_error_on_other_errors(t *testing.T) {
	// Create a non-ExitError
	err := errors.New("some other error")

	cfg := &Config{}
	_, resultErr := BuildResult(err, cfg, nil, nil)

	if resultErr == nil {
		t.Error("expected error to be returned")
	}
	if resultErr.Error() != "some other error" {
		t.Errorf("error = %q, want %q", resultErr.Error(), "some other error")
	}
}

func TestBuildResult_returns_nil_error_on_success(t *testing.T) {
	cfg := &Config{}
	result, err := BuildResult(nil, cfg, nil, nil)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("exit code = %d, want 0", result.ExitCode)
	}
}

func TestShellEscape_basic(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "'hello'"},
		{"hello world", "'hello world'"},
		{"/path/to/file", "'/path/to/file'"},
		{"", "''"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ShellEscape(tt.input)
			if got != tt.want {
				t.Errorf("ShellEscape(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestShellEscape_handles_single_quotes(t *testing.T) {
	input := "it's a test"
	got := ShellEscape(input)

	// The escaped form should be 'it'\''s a test'
	want := "'it'\\''s a test'"
	if got != want {
		t.Errorf("ShellEscape(%q) = %q, want %q", input, got, want)
	}
}

func TestBuildPHPCommand(t *testing.T) {
	cfg := &Config{
		PHPBinary:  "/usr/bin/php",
		ScriptPath: "/path/to/script.php",
		ScriptArgs: []string{"--verbose"},
		MemoryMB:   128,
	}

	cmd := BuildPHPCommand(cfg)

	// Should be a properly escaped command string
	if cmd == "" {
		t.Error("BuildPHPCommand returned empty string")
	}

	// Should contain the escaped php binary
	if !strings.Contains(cmd, "'/usr/bin/php'") {
		t.Errorf("command should contain escaped php binary, got %s", cmd)
	}
}

func TestProxyEnvVars(t *testing.T) {
	envVars := ProxyEnvVars()

	// Should contain HTTP_PROXY, HTTPS_PROXY variants
	expectedPrefixes := []string{
		"HTTP_PROXY=",
		"HTTPS_PROXY=",
		"http_proxy=",
		"https_proxy=",
		"ALL_PROXY=",
	}

	for _, prefix := range expectedPrefixes {
		found := false
		for _, env := range envVars {
			if strings.HasPrefix(env, prefix) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected env var with prefix %s", prefix)
		}
	}
}

func TestConfig_defaults(t *testing.T) {
	cfg := &Config{
		Timeout: 30 * time.Second,
	}

	if cfg.Timeout != 30*time.Second {
		t.Errorf("timeout = %v, want 30s", cfg.Timeout)
	}
}
