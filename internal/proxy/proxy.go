package proxy

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// SandboxBridgePort is the port used by socat to bridge network traffic
// from inside the sandbox to the proxy's Unix socket.
const SandboxBridgePort = 19850

// Proxy is an HTTP/HTTPS proxy with domain filtering.
type Proxy struct {
	Filter     *DomainFilter
	Port       int
	SocketPath string // Unix socket path (for Linux sandbox)
	Verbose    bool
	listener   net.Listener
	server     *http.Server
	wg         sync.WaitGroup
}

// NewProxy creates a new proxy server.
func NewProxy(filter *DomainFilter) *Proxy {
	return &Proxy{
		Filter: filter,
		Port:   0, // Will be assigned when started
	}
}

// Start starts the proxy server on an available TCP port.
func (p *Proxy) Start() error {
	var err error
	p.listener, err = net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("failed to start proxy listener: %w", err)
	}

	// Get the assigned port
	addr := p.listener.Addr().(*net.TCPAddr)
	p.Port = addr.Port

	p.server = &http.Server{
		Handler: http.HandlerFunc(p.handleRequest),
	}

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		if err := p.server.Serve(p.listener); err != http.ErrServerClosed {
			if p.Verbose {
				fmt.Fprintf(os.Stderr, "[proxy] Server error: %v\n", err)
			}
		}
	}()

	if p.Verbose {
		fmt.Fprintf(os.Stderr, "[proxy] Started on port %d\n", p.Port)
	}

	return nil
}

// StartUnix starts the proxy server on a Unix domain socket.
func (p *Proxy) StartUnix(socketPath string) error {
	// Remove existing socket file if present
	_ = os.Remove(socketPath)

	var err error
	p.listener, err = net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("failed to start proxy on unix socket: %w", err)
	}

	p.SocketPath = socketPath

	// Set restrictive permissions - only owner can access
	if err := os.Chmod(socketPath, 0600); err != nil {
		fmt.Fprintf(os.Stderr, "[proxy] Warning: could not set socket permissions: %v\n", err)
	}

	p.server = &http.Server{
		Handler: http.HandlerFunc(p.handleRequest),
	}

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		if err := p.server.Serve(p.listener); err != http.ErrServerClosed {
			if p.Verbose {
				fmt.Fprintf(os.Stderr, "[proxy] Server error: %v\n", err)
			}
		}
	}()

	if p.Verbose {
		fmt.Fprintf(os.Stderr, "[proxy] Started on unix socket %s\n", socketPath)
	}

	return nil
}

// Stop stops the proxy server.
func (p *Proxy) Stop() error {
	if p.server == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := p.server.Shutdown(ctx)
	p.wg.Wait()

	// Clean up Unix socket if used
	if p.SocketPath != "" {
		_ = os.Remove(p.SocketPath)
	}

	if p.Verbose {
		fmt.Fprintln(os.Stderr, "[proxy] Stopped")
	}

	return err
}

// Address returns the proxy address (127.0.0.1:port).
func (p *Proxy) Address() string {
	return fmt.Sprintf("127.0.0.1:%d", p.Port)
}

// EnvVarsWithSOCKS5 returns env vars including SOCKS5 proxy address.
func (p *Proxy) EnvVarsWithSOCKS5(socks5Addr string) []string {
	addr := p.Address()
	vars := []string{
		fmt.Sprintf("HTTP_PROXY=http://%s", addr),
		fmt.Sprintf("HTTPS_PROXY=http://%s", addr),
		fmt.Sprintf("http_proxy=http://%s", addr),
		fmt.Sprintf("https_proxy=http://%s", addr),
	}
	if socks5Addr != "" {
		vars = append(vars, fmt.Sprintf("ALL_PROXY=socks5://%s", socks5Addr))
		vars = append(vars, fmt.Sprintf("all_proxy=socks5://%s", socks5Addr))
	}
	return vars
}

// handleRequest handles incoming proxy requests.
func (p *Proxy) handleRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		p.handleConnect(w, r)
	} else {
		p.handleHTTP(w, r)
	}
}

// handleConnect handles HTTPS CONNECT requests (tunneling).
func (p *Proxy) handleConnect(w http.ResponseWriter, r *http.Request) {
	host := r.Host

	// Check if domain is allowed
	if !p.Filter.IsAllowed(host) {
		if p.Verbose {
			fmt.Fprintf(os.Stderr, "[proxy] BLOCKED: %s\n", host)
		}
		http.Error(w, fmt.Sprintf("Domain not allowed: %s", host), http.StatusForbidden)
		return
	}

	if p.Verbose {
		fmt.Fprintf(os.Stderr, "[proxy] CONNECT: %s\n", host)
	}

	// Ensure host has port
	if !strings.Contains(host, ":") {
		host = host + ":443"
	}

	// Connect to target
	targetConn, err := net.DialTimeout("tcp", host, 10*time.Second)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	// Hijack the connection
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		_ = targetConn.Close()
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		_ = targetConn.Close()
		return
	}

	// Send 200 OK to client
	_, _ = clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	// Bidirectional copy
	go func() {
		_, _ = io.Copy(targetConn, clientConn)
		_ = targetConn.Close()
	}()
	go func() {
		_, _ = io.Copy(clientConn, targetConn)
		_ = clientConn.Close()
	}()
}

// handleHTTP handles regular HTTP proxy requests.
func (p *Proxy) handleHTTP(w http.ResponseWriter, r *http.Request) {
	host := r.Host
	if host == "" {
		host = r.URL.Host
	}

	// Check if domain is allowed
	if !p.Filter.IsAllowed(host) {
		if p.Verbose {
			fmt.Fprintf(os.Stderr, "[proxy] BLOCKED: %s %s\n", r.Method, r.URL)
		}
		http.Error(w, fmt.Sprintf("Domain not allowed: %s", host), http.StatusForbidden)
		return
	}

	if p.Verbose {
		fmt.Fprintf(os.Stderr, "[proxy] %s %s\n", r.Method, r.URL)
	}

	// Create outgoing request
	outReq := &http.Request{
		Method: r.Method,
		URL:    r.URL,
		Header: r.Header.Clone(),
		Body:   r.Body,
	}

	// Remove hop-by-hop headers
	outReq.Header.Del("Proxy-Connection")
	outReq.Header.Del("Proxy-Authenticate")
	outReq.Header.Del("Proxy-Authorization")

	// Make request
	client := &http.Client{
		Timeout: 60 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Don't follow redirects, let the client handle them
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Do(outReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer func() { _ = resp.Body.Close() }()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Write status and body
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}
