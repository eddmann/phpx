package proxy

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// Manager coordinates HTTP and SOCKS5 proxy servers for sandboxed execution.
type Manager struct {
	httpProxy   *Proxy
	socks5Proxy *SOCKS5Proxy
	socketProxy *Proxy
	socketPath  string
	verbose     bool
}

// ManagerConfig holds configuration for the proxy manager.
type ManagerConfig struct {
	AllowedHosts []string
	Verbose      bool
}

// NewManager creates and starts all necessary proxy servers.
// Returns nil if no proxies are needed.
func NewManager(cfg ManagerConfig) (*Manager, error) {
	m := &Manager{verbose: cfg.Verbose}

	// Create filter
	filter := NewDomainFilter()
	if len(cfg.AllowedHosts) == 0 {
		filter.AllowAll()
	} else {
		for _, host := range cfg.AllowedHosts {
			filter.AddAllowed(host)
		}
	}

	// Start HTTP proxy
	m.httpProxy = NewProxy(filter)
	m.httpProxy.Verbose = cfg.Verbose
	if err := m.httpProxy.Start(); err != nil {
		return nil, fmt.Errorf("failed to start HTTP proxy: %w", err)
	}

	// Start SOCKS5 proxy for non-HTTP traffic
	m.socks5Proxy = NewSOCKS5Proxy(filter)
	m.socks5Proxy.Verbose = cfg.Verbose
	if err := m.socks5Proxy.Start(); err != nil {
		// Warn but continue - SOCKS5 is optional
		fmt.Fprintf(os.Stderr, "[phpx] Warning: SOCKS5 proxy failed to start: %v (non-HTTP traffic may fail)\n", err)
		m.socks5Proxy = nil
	}

	// On Linux, create Unix socket for sandbox isolation
	if runtime.GOOS == "linux" {
		socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("phpx-proxy-%s.sock", randomID(8)))
		m.socketProxy = NewProxy(filter)
		m.socketProxy.Verbose = cfg.Verbose
		if err := m.socketProxy.StartUnix(socketPath); err != nil {
			if cfg.Verbose {
				fmt.Fprintf(os.Stderr, "[phpx] Warning: Could not start Unix socket proxy: %v\n", err)
			}
			m.socketProxy = nil
		} else {
			m.socketPath = socketPath
		}
	}

	return m, nil
}

// Stop shuts down all proxy servers.
func (m *Manager) Stop() {
	if m.socketProxy != nil {
		_ = m.socketProxy.Stop()
	}
	if m.socks5Proxy != nil {
		_ = m.socks5Proxy.Stop()
	}
	if m.httpProxy != nil {
		_ = m.httpProxy.Stop()
	}
}

// Port returns the HTTP proxy port.
func (m *Manager) Port() int {
	if m.httpProxy != nil {
		return m.httpProxy.Port
	}
	return 0
}

// SOCKS5Port returns the SOCKS5 proxy port.
func (m *Manager) SOCKS5Port() int {
	if m.socks5Proxy != nil {
		return m.socks5Proxy.Port
	}
	return 0
}

// SocketPath returns the Unix socket path (Linux only).
func (m *Manager) SocketPath() string {
	return m.socketPath
}

// EnvVars returns environment variables for configuring proxy in subprocesses.
func (m *Manager) EnvVars() []string {
	if m.httpProxy == nil {
		return nil
	}
	socks5Addr := ""
	if m.socks5Proxy != nil {
		socks5Addr = m.socks5Proxy.Address()
	}
	return m.httpProxy.EnvVarsWithSOCKS5(socks5Addr)
}

// randomID generates a cryptographically random ID for temp file naming.
func randomID(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		// Fallback to less secure but still unique naming
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
