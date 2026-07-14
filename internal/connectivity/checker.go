package connectivity

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	appconfig "github.com/nibra180/stargate-tui/internal/config"
)

const (
	DefaultTimeout     = 2 * time.Second
	DefaultConcurrency = 8
)

type contextDialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// Checker performs bounded, direct TCP connectivity checks without logging in.
type Checker struct {
	timeout   time.Duration
	semaphore chan struct{}
	dialer    contextDialer
}

// New creates a TCP checker with the given per-host timeout and concurrency.
func New(timeout time.Duration, concurrency int) *Checker {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	if concurrency <= 0 {
		concurrency = DefaultConcurrency
	}
	return &Checker{
		timeout:   timeout,
		semaphore: make(chan struct{}, concurrency),
		dialer:    &net.Dialer{},
	}
}

// Check tests whether host's configured address accepts a direct TCP connection.
func (c *Checker) Check(host appconfig.Host) error {
	if c == nil {
		return errors.New("connectivity checker is nil")
	}
	hostName := strings.TrimSpace(host.HostName)
	port := strings.TrimSpace(host.Port)
	if hostName == "" || port == "" {
		return errors.New("host name and port are required")
	}

	c.semaphore <- struct{}{}
	defer func() { <-c.semaphore }()

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	address := net.JoinHostPort(hostName, port)
	connection, err := c.dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("connect to %s: %w", address, err)
	}
	_ = connection.Close()
	return nil
}
