package connectivity

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	appconfig "github.com/warikoda/stargate/internal/config"
)

func TestCheckDialsConfiguredHostAndPort(t *testing.T) {
	t.Parallel()

	dialer := &stubDialer{}
	checker := New(time.Second, 1)
	checker.dialer = dialer

	err := checker.Check(appconfig.Host{HostName: "2001:db8::10", Port: "2200"})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if dialer.network != "tcp" || dialer.address != "[2001:db8::10]:2200" {
		t.Errorf("dialed %s %s, want tcp [2001:db8::10]:2200", dialer.network, dialer.address)
	}
}

func TestCheckReturnsDialAndValidationErrors(t *testing.T) {
	t.Parallel()

	checker := New(time.Second, 1)
	checker.dialer = &stubDialer{err: errors.New("connection refused")}
	if err := checker.Check(appconfig.Host{HostName: "example.com", Port: "22"}); err == nil {
		t.Fatal("Check() returned nil for a dial error")
	}
	if err := checker.Check(appconfig.Host{}); err == nil {
		t.Fatal("Check() returned nil for an empty endpoint")
	}
}

type stubDialer struct {
	network string
	address string
	err     error
}

func (d *stubDialer) DialContext(_ context.Context, network, address string) (net.Conn, error) {
	d.network = network
	d.address = address
	if d.err != nil {
		return nil, d.err
	}

	client, server := net.Pipe()
	_ = server.Close()
	return client, nil
}
