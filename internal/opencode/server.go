package opencode

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"time"
)

// Server manages a child OpenCode server process
type Server struct {
	cmd        *exec.Cmd
	port       int
	projectDir string
	url        string
	verbose    bool
}

// ServerConfig holds configuration for the managed server
type ServerConfig struct {
	ProjectDir string
	Port       int // 0 means auto-select
	Verbose    bool
}

// NewServer creates a new managed OpenCode server
func NewServer(cfg ServerConfig) *Server {
	port := cfg.Port
	if port == 0 {
		port = findFreePort()
	}

	return &Server{
		port:       port,
		projectDir: cfg.ProjectDir,
		url:        fmt.Sprintf("http://127.0.0.1:%d", port),
		verbose:    cfg.Verbose,
	}
}

// Start launches the OpenCode server process
func (s *Server) Start(ctx context.Context) error {
	if s.cmd != nil {
		return fmt.Errorf("server already running")
	}

	// Build the command
	s.cmd = exec.CommandContext(ctx, "opencode", "serve", "--port", fmt.Sprintf("%d", s.port))
	s.cmd.Dir = s.projectDir

	// Suppress output unless verbose
	if s.verbose {
		s.cmd.Stdout = os.Stdout
		s.cmd.Stderr = os.Stderr
	}

	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start opencode server: %w", err)
	}

	// Wait for server to be ready
	if err := s.waitForReady(ctx, 30*time.Second); err != nil {
		if stopErr := s.Stop(); stopErr != nil {
			return fmt.Errorf("server failed to start: %w (also failed to stop: %v)", err, stopErr)
		}
		return fmt.Errorf("server failed to start: %w", err)
	}

	return nil
}

// Stop terminates the OpenCode server process
func (s *Server) Stop() error {
	if s.cmd == nil || s.cmd.Process == nil {
		return nil
	}

	// Send SIGTERM first
	if err := s.cmd.Process.Signal(os.Interrupt); err != nil {
		// If SIGTERM fails, force kill
		if killErr := s.cmd.Process.Kill(); killErr != nil {
			return fmt.Errorf("failed to kill process: %w", killErr)
		}
	}

	// Wait for process to exit with timeout
	done := make(chan error, 1)
	go func() {
		done <- s.cmd.Wait()
	}()

	select {
	case <-done:
		// Process exited
	case <-time.After(5 * time.Second):
		// Force kill if still running
		if err := s.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to force kill process: %w", err)
		}
	}

	s.cmd = nil
	return nil
}

// URL returns the server URL
func (s *Server) URL() string {
	return s.url
}

// Port returns the server port
func (s *Server) Port() int {
	return s.port
}

// IsRunning checks if the server process is still running
func (s *Server) IsRunning() bool {
	if s.cmd == nil || s.cmd.Process == nil {
		return false
	}
	// Check if process is still alive
	return s.cmd.ProcessState == nil
}

// waitForReady polls the server until it responds or timeout
func (s *Server) waitForReady(ctx context.Context, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Try to connect
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", s.port), time.Second)
		if err == nil {
			if closeErr := conn.Close(); closeErr != nil {
				return fmt.Errorf("failed to close connection: %w", closeErr)
			}
			// Give the server a moment to fully initialize
			time.Sleep(500 * time.Millisecond)
			return nil
		}

		time.Sleep(200 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for server to be ready")
}

// findFreePort finds an available port
func findFreePort() int {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 8766 // Fallback to default
	}
	defer func() {
		if err := listener.Close(); err != nil {
			// Log error but continue - port was already obtained
			fmt.Printf("Warning: failed to close listener: %v\n", err)
		}
	}()
	return listener.Addr().(*net.TCPAddr).Port
}
