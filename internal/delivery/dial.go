package delivery

import (
	"context"
	"fmt"
	"net"
	"time"

	"webhookrelay/internal/ssrf"
)

// dialTimeout bounds how long establishing the TCP connection itself may
// take, separate from the overall request timeout.
const dialTimeout = 5 * time.Second

// safeDialContext resolves the destination host itself — rather than
// letting net.Dialer resolve it internally — rejects the connection if any
// resolved address is blocked (internal/ssrf), and then dials that exact
// validated IP directly.
//
// Doing the resolution and the dial as one step, against the specific IP we
// just validated rather than the hostname again, closes the DNS-rebinding
// gap: there is no second resolution between "checked" and "connected" for
// an attacker to swap out.
//
// Because this runs as the Transport's DialContext, it also transparently
// covers HTTP redirects — following a redirect to a new host dials through
// this same function again, so a redirect can't be used to reach a blocked
// address either.
func safeDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("invalid address %q: %w", addr, err)
	}

	var ips []net.IP
	if ip := net.ParseIP(host); ip != nil {
		ips = []net.IP{ip}
	} else {
		ips, err = net.DefaultResolver.LookupIP(ctx, "ip", host)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve host %q: %w", host, err)
		}
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("no addresses found for host %q", host)
	}

	for _, ip := range ips {
		if ssrf.IsBlocked(ip) {
			return nil, fmt.Errorf("%w: %s resolves to %s", ssrf.ErrBlockedAddress, host, ip)
		}
	}

	dialer := &net.Dialer{Timeout: dialTimeout}
	return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].String(), port))
}
