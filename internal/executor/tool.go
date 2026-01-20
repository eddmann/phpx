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

// ToolOptions holds options for running a tool.
type ToolOptions struct {
	// Tool settings
	PHPBinary  string
	ToolDir    string // Directory where tool is installed
	BinaryName string // Name of the binary to run

	// Sandbox options
	Sandbox        sandbox.Sandbox
	Network        bool
	AllowedHosts   []string
	AllowedEnvVars []string
	ReadPaths      []string
	WritePaths     []string
	MemoryMB       int
	Timeout        time.Duration
	CPUSeconds     int

	// Tool arguments
	Args []string

	// Working directory (where tool runs from)
	WorkDir string

	// I/O streams - if set, streams directly instead of buffering
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	// Output
	Verbose bool
}

// ToolRunner executes Composer tools with optional sandboxing.
type ToolRunner struct {
	opts *ToolOptions
}

// NewToolRunner creates a new tool runner.
func NewToolRunner(opts *ToolOptions) *ToolRunner {
	return &ToolRunner{opts: opts}
}

// Run executes the tool.
func (r *ToolRunner) Run(ctx context.Context) (*sandbox.Result, error) {
	sb := r.opts.Sandbox

	// Construct path to tool binary
	binaryPath := filepath.Join(r.opts.ToolDir, "vendor", "bin", r.opts.BinaryName)

	// Start proxy if network is needed and we're sandboxing
	var proxyMgr *proxy.Manager
	var proxyEnv []string
	var proxySocketPath string
	var proxyPort int
	var proxySOCKS5Port int

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
		proxySOCKS5Port = proxyMgr.SOCKS5Port()
	}

	// Determine working directory
	workDir := r.opts.WorkDir
	if workDir == "" {
		var err error
		workDir, err = os.Getwd()
		if err != nil {
			workDir = "/"
		}
	}

	// Add tool directory and current working directory to readable paths
	readPaths := append(r.opts.ReadPaths, r.opts.ToolDir)
	readPaths = append(readPaths, workDir)

	// Tools often need to write to current directory
	writePaths := append(r.opts.WritePaths, workDir)

	// Prepare sandbox config
	sandboxCfg := &sandbox.Config{
		Network:         r.opts.Network,
		AllowedHosts:    r.opts.AllowedHosts,
		ProxySocketPath: proxySocketPath,
		ProxyPort:       proxyPort,
		ProxySOCKS5Port: proxySOCKS5Port,
		ReadablePaths:   readPaths,
		WritablePaths:   writePaths,
		MemoryMB:        r.opts.MemoryMB,
		Timeout:         r.opts.Timeout,
		CPUSeconds:      r.opts.CPUSeconds,
		PHPBinary:       r.opts.PHPBinary,
		AutoloadFile:    "", // Tools use their own autoloading
		ScriptPath:      binaryPath,
		ScriptArgs:      r.opts.Args,
		WorkDir:         workDir,
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

	if r.opts.Verbose {
		fmt.Fprintf(os.Stderr, "[phpx] Running tool: %s\n", binaryPath)
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
