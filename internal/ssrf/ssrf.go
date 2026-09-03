// Package ssrf defines which destination addresses outbound webhook
// deliveries are never allowed to reach (doc section 21).
package ssrf

import (
	"context"
	"errors"
	"fmt"
	"net"
)

// ErrBlockedAddress is returned when a destination is (or resolves to) an
// address outbound deliveries must never reach.
var ErrBlockedAddress = errors.New("destination address is not allowed")

// blockedNets covers loopback, the private RFC 1918 ranges, and link-local
// (which includes the 169.254.169.254 cloud metadata address every major
// cloud provider uses) — plus their IPv6 equivalents.
var blockedNets = mustParseCIDRs(
	"127.0.0.0/8",
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"169.254.0.0/16",
	"::1/128",
	"fc00::/7",
	"fe80::/10",
)

func mustParseCIDRs(cidrs ...string) []*net.IPNet {
	nets := make([]*net.IPNet, 0, len(cidrs))
	for _, c := range cidrs {
		_, n, err := net.ParseCIDR(c)
		if err != nil {
			panic(fmt.Sprintf("ssrf: invalid CIDR %q: %v", c, err))
		}
		nets = append(nets, n)
	}
	return nets
}

// IsBlocked reports whether ip falls within a range outbound deliveries must
// never reach.
func IsBlocked(ip net.IP) bool {
	if ip.IsUnspecified() { // 0.0.0.0 / ::
		return true
	}
	for _, n := range blockedNets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// CheckHost resolves host (which may already be a literal IP) and reports
// an error if it, or any address it resolves to, is blocked.
//
// This is a coarse, early check meant for endpoint registration — good UX,
// rejecting an obviously bad URL immediately. It is NOT sufficient
// protection on its own: DNS can change between registration and actual
// delivery (DNS rebinding). The real enforcement point is the dial hook in
// internal/delivery, which validates the address actually being connected
// to at delivery time, with no second resolution in between.
func CheckHost(ctx context.Context, host string) error {
	if ip := net.ParseIP(host); ip != nil {
		if IsBlocked(ip) {
			return fmt.Errorf("%w: %s", ErrBlockedAddress, ip)
		}
		return nil
	}

	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
	if err != nil {
		return fmt.Errorf("failed to resolve host %q: %w", host, err)
	}
	for _, ip := range ips {
		if IsBlocked(ip) {
			return fmt.Errorf("%w: %s resolves to %s", ErrBlockedAddress, host, ip)
		}
	}
	return nil
}
