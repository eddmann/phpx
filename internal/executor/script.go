package executor

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/eddmann/phpx/internal/proxy"
	"github.com/eddmann/phpx/internal/sandbox"
)

// ScriptOptions holds options for running a script.
type ScriptOptions struct {
	// Script path
	ScriptPath string

	// PHP settings
	PHPBinary    string
	AutoloadFile string

	// Sandbox options
	Sandbox        sandbox.Sandbox
	Network        bool
	AllowedHosts   []string
	AllowedEnvVars []string // Additional env vars to pass through (--allow-env flag)
	ReadPaths      []string
	WritePaths     []string
	MemoryMB       int
	Timeout        time.Duration
	CPUSeconds     int

	// Script arguments
	Args []string

	// I/O streams - if set, streams directly instead of buffering
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	// Output
	Verbose bool
}

// ScriptRunner executes PHP scripts with optional sandboxing.
type ScriptRunner struct {
	opts *ScriptOptions
}

// NewScriptRunner creates a new script runner.
func NewScriptRunner(opts *ScriptOptions) *ScriptRunner {
	return &ScriptRunner{opts: opts}
}

// Run executes the script.
func (r *ScriptRunner) Run(ctx context.Context) (*sandbox.Result, error) {
	sb := r.opts.Sandbox

	// Start proxy if network is needed and we're sandboxing
	var proxyMgr *proxy.Manager
	var proxyEnv []string
	var proxySocketPath string
	var proxyPort int

	needsProxy := sb.IsSandboxed() && r.opts.Network

	if needsProxy {
		var err error
		proxyMgr, err = proxy.NewManager(proxy.ManagerConfig{
			AllowedHosts: r.opts.AllowedHosts,
			Verbose:      r.opts.Verbose,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to start proxy: %w", err)
		}
		defer proxyMgr.Stop()

		proxyEnv = proxyMgr.EnvVars()
		proxySocketPath = proxyMgr.SocketPath()
		proxyPort = proxyMgr.Port()
	}

	// Prepare sandbox config
	sandboxCfg := &sandbox.Config{
		Network:         r.opts.Network,
		AllowedHosts:    r.opts.AllowedHosts,
		ProxySocketPath: proxySocketPath,
		ProxyPort:       proxyPort,
		ReadablePaths:   r.opts.ReadPaths,
		WritablePaths:   r.opts.WritePaths,
		MemoryMB:        r.opts.MemoryMB,
		Timeout:         r.opts.Timeout,
		CPUSeconds:      r.opts.CPUSeconds,
		PHPBinary:       r.opts.PHPBinary,
		AutoloadFile:    r.opts.AutoloadFile,
		ScriptPath:      r.opts.ScriptPath,
		ScriptArgs:      r.opts.Args,
		WorkDir:         filepath.Dir(r.opts.ScriptPath),
		Env:             proxyEnv,
		AllowedEnvVars:  r.opts.AllowedEnvVars,
		Stdin:           r.opts.Stdin,
		Stdout:          r.opts.Stdout,
		Stderr:          r.opts.Stderr,
		Verbose:         r.opts.Verbose,
	}

	if r.opts.Verbose && sb.IsSandboxed() {
		fmt.Fprintf(os.Stderr, "[phpx] Using sandbox: %s\n", sb.Name())
	}

	// Execute
	if r.opts.Verbose {
		fmt.Fprintf(os.Stderr, "[phpx] Running %s\n", r.opts.ScriptPath)
	}

	// Create execution context with timeout
	execCtx := ctx
	if r.opts.Timeout > 0 {
		var cancel context.CancelFunc
		execCtx, cancel = context.WithTimeout(ctx, r.opts.Timeout)
		defer cancel()
	}

	result, err := sb.Execute(execCtx, sandboxCfg)
	if err != nil {
		return result, fmt.Errorf("execution failed: %w", err)
	}

	return result, nil
}
