package ssrf

import (
	"net"
	"testing"
)

func TestIsBlocked(t *testing.T) {
	cases := []struct {
		name string
		ip   string
		want bool
	}{
		{"loopback v4", "127.0.0.1", true},
		{"loopback v4 alt form", "127.1.2.3", true},
		{"private 10/8", "10.0.0.5", true},
		{"private 172.16/12", "172.20.0.1", true},
		{"private 192.168/16", "192.168.1.1", true},
		{"link-local incl. cloud metadata", "169.254.169.254", true},
		{"unspecified v4", "0.0.0.0", true},
		{"loopback v6", "::1", true},
		{"unique local v6", "fc00::1", true},
		{"link-local v6", "fe80::1", true},
		{"unspecified v6", "::", true},
		{"public v4", "8.8.8.8", false},
		{"public v4 2", "93.184.216.34", false},
		{"public v6", "2606:4700:4700::1111", false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ip := net.ParseIP(c.ip)
			if ip == nil {
				t.Fatalf("test case has invalid IP %q", c.ip)
			}
			if got := IsBlocked(ip); got != c.want {
				t.Errorf("IsBlocked(%s) = %v, want %v", c.ip, got, c.want)
			}
		})
	}
}

func TestCheckHost_LiteralIP(t *testing.T) {
	ctx := t.Context()

	if err := CheckHost(ctx, "127.0.0.1"); err == nil {
		t.Error("CheckHost(127.0.0.1) = nil, want error")
	}
	if err := CheckHost(ctx, "169.254.169.254"); err == nil {
		t.Error("CheckHost(169.254.169.254) = nil, want error")
	}
	if err := CheckHost(ctx, "8.8.8.8"); err != nil {
		t.Errorf("CheckHost(8.8.8.8) = %v, want nil", err)
	}
}

func TestCheckHost_Localhost(t *testing.T) {
	// "localhost" resolves to a loopback address on essentially every
	// system, so it should be blocked without any special-case string
	// matching — resolution + IsBlocked is enough.
	if err := CheckHost(t.Context(), "localhost"); err == nil {
		t.Error("CheckHost(localhost) = nil, want error")
	}
}
