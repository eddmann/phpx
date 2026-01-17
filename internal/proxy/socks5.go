package proxy

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
)

// SOCKS5 protocol constants.
const (
	socks5Version = 0x05

	// Authentication methods
	authNone     = 0x00
	authNoAccept = 0xFF

	// Commands
	cmdConnect = 0x01

	// Address types
	atypIPv4   = 0x01
	atypDomain = 0x03
	atypIPv6   = 0x04

	// Reply codes
	repSuccess          = 0x00
	repGeneralFailure   = 0x01
	repConnNotAllowed   = 0x02
	repNetUnreachable   = 0x03
	repHostUnreachable  = 0x04
	repConnRefused      = 0x05
	repTTLExpired       = 0x06
	repCmdNotSupported  = 0x07
	repAddrNotSupported = 0x08
)

// SOCKS5Proxy is a SOCKS5 proxy server with domain filtering.
type SOCKS5Proxy struct {
	Filter     *DomainFilter
	Port       int
	SocketPath string
	Verbose    bool
	listener   net.Listener
	wg         sync.WaitGroup
	done       chan struct{}
}

// NewSOCKS5Proxy creates a new SOCKS5 proxy server.
func NewSOCKS5Proxy(filter *DomainFilter) *SOCKS5Proxy {
	return &SOCKS5Proxy{
		Filter: filter,
		done:   make(chan struct{}),
	}
}

// Start starts the SOCKS5 proxy on an available TCP port.
func (s *SOCKS5Proxy) Start() error {
	var err error
	s.listener, err = net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("failed to start SOCKS5 listener: %w", err)
	}

	addr := s.listener.Addr().(*net.TCPAddr)
	s.Port = addr.Port

	s.wg.Add(1)
	go s.acceptLoop()

	if s.Verbose {
		fmt.Fprintf(os.Stderr, "[socks5] Started on port %d\n", s.Port)
	}

	return nil
}

// Stop stops the SOCKS5 proxy.
func (s *SOCKS5Proxy) Stop() error {
	close(s.done)
	if s.listener != nil {
		_ = s.listener.Close()
	}
	s.wg.Wait()

	if s.SocketPath != "" {
		_ = os.Remove(s.SocketPath)
	}

	if s.Verbose {
		fmt.Fprintln(os.Stderr, "[socks5] Stopped")
	}

	return nil
}

// Address returns the proxy address.
func (s *SOCKS5Proxy) Address() string {
	return fmt.Sprintf("127.0.0.1:%d", s.Port)
}

func (s *SOCKS5Proxy) acceptLoop() {
	defer s.wg.Done()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.done:
				return
			default:
				if s.Verbose {
					fmt.Fprintf(os.Stderr, "[socks5] Accept error: %v\n", err)
				}
				continue
			}
		}

		go s.handleConnection(conn)
	}
}

func (s *SOCKS5Proxy) handleConnection(conn net.Conn) {
	defer func() { _ = conn.Close() }()

	// Set deadline for handshake
	_ = conn.SetDeadline(time.Now().Add(30 * time.Second))

	// Read greeting
	if err := s.handleGreeting(conn); err != nil {
		if s.Verbose {
			fmt.Fprintf(os.Stderr, "[socks5] Greeting error: %v\n", err)
		}
		return
	}

	// Handle request
	if err := s.handleRequest(conn); err != nil {
		if s.Verbose {
			fmt.Fprintf(os.Stderr, "[socks5] Request error: %v\n", err)
		}
		return
	}
}

func (s *SOCKS5Proxy) handleGreeting(conn net.Conn) error {
	// Read version and number of methods
	header := make([]byte, 2)
	if _, err := io.ReadFull(conn, header); err != nil {
		return err
	}

	if header[0] != socks5Version {
		return fmt.Errorf("unsupported SOCKS version: %d", header[0])
	}

	// Read methods
	numMethods := int(header[1])
	methods := make([]byte, numMethods)
	if _, err := io.ReadFull(conn, methods); err != nil {
		return err
	}

	// Check for no-auth method
	hasNoAuth := false
	for _, m := range methods {
		if m == authNone {
			hasNoAuth = true
			break
		}
	}

	if !hasNoAuth {
		_, _ = conn.Write([]byte{socks5Version, authNoAccept})
		return fmt.Errorf("no acceptable auth method")
	}

	// Accept no-auth
	_, err := conn.Write([]byte{socks5Version, authNone})
	return err
}

func (s *SOCKS5Proxy) handleRequest(conn net.Conn) error {
	// Read request header
	header := make([]byte, 4)
	if _, err := io.ReadFull(conn, header); err != nil {
		return err
	}

	if header[0] != socks5Version {
		return fmt.Errorf("unsupported SOCKS version: %d", header[0])
	}

	if header[1] != cmdConnect {
		_ = s.sendReply(conn, repCmdNotSupported, nil)
		return fmt.Errorf("unsupported command: %d", header[1])
	}

	// Parse destination address
	var host string

	switch header[3] {
	case atypIPv4:
		addr := make([]byte, 4)
		if _, err := io.ReadFull(conn, addr); err != nil {
			return err
		}
		host = net.IP(addr).String()

	case atypDomain:
		lenByte := make([]byte, 1)
		if _, err := io.ReadFull(conn, lenByte); err != nil {
			return err
		}
		domain := make([]byte, lenByte[0])
		if _, err := io.ReadFull(conn, domain); err != nil {
			return err
		}
		host = string(domain)

	case atypIPv6:
		addr := make([]byte, 16)
		if _, err := io.ReadFull(conn, addr); err != nil {
			return err
		}
		host = net.IP(addr).String()

	default:
		_ = s.sendReply(conn, repAddrNotSupported, nil)
		return fmt.Errorf("unsupported address type: %d", header[3])
	}

	// Read port
	portBytes := make([]byte, 2)
	if _, err := io.ReadFull(conn, portBytes); err != nil {
		return err
	}
	port := binary.BigEndian.Uint16(portBytes)

	// Check domain filter
	hostWithPort := host + ":" + strconv.Itoa(int(port))
	if !s.Filter.IsAllowed(hostWithPort) {
		if s.Verbose {
			fmt.Fprintf(os.Stderr, "[socks5] BLOCKED: %s\n", hostWithPort)
		}
		_ = s.sendReply(conn, repConnNotAllowed, nil)
		return fmt.Errorf("host not allowed: %s", host)
	}

	if s.Verbose {
		fmt.Fprintf(os.Stderr, "[socks5] CONNECT: %s\n", hostWithPort)
	}

	// Connect to target
	target, err := net.DialTimeout("tcp", hostWithPort, 10*time.Second)
	if err != nil {
		if s.Verbose {
			fmt.Fprintf(os.Stderr, "[socks5] Connect failed: %v\n", err)
		}
		_ = s.sendReply(conn, repHostUnreachable, nil)
		return err
	}
	defer func() { _ = target.Close() }()

	// Send success reply
	localAddr, ok := target.LocalAddr().(*net.TCPAddr)
	if !ok {
		return fmt.Errorf("target connection is not TCP")
	}
	if err := s.sendReply(conn, repSuccess, localAddr); err != nil {
		return err
	}

	// Clear deadline for relay
	_ = conn.SetDeadline(time.Time{})

	// Relay data bidirectionally
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		_, _ = io.Copy(target, conn)
		if tcpTarget, ok := target.(*net.TCPConn); ok {
			_ = tcpTarget.CloseWrite()
		}
	}()

	go func() {
		defer wg.Done()
		_, _ = io.Copy(conn, target)
		if tcpConn, ok := conn.(*net.TCPConn); ok {
			_ = tcpConn.CloseWrite()
		}
	}()

	wg.Wait()
	return nil
}

func (s *SOCKS5Proxy) sendReply(conn net.Conn, rep byte, addr *net.TCPAddr) error {
	reply := make([]byte, 10)
	reply[0] = socks5Version
	reply[1] = rep
	reply[2] = 0x00 // reserved
	reply[3] = atypIPv4

	if addr != nil {
		ip := addr.IP.To4()
		if ip == nil {
			ip = net.IPv4zero
		}
		copy(reply[4:8], ip)
		binary.BigEndian.PutUint16(reply[8:10], uint16(addr.Port))
	}

	_, err := conn.Write(reply)
	return err
}
