package media

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"
)

// allowInsecureImageBedDialsForTesting temporarily replaces the image bed
// dial function with a plain loopback-capable dialer. Tests that exercise
// the genericImageBedUploader against an httptest.NewServer must call this
// because httptest binds to 127.0.0.1, which the production SSRF guard
// (correctly) refuses to contact. The hook is restored by t.Cleanup.
func allowInsecureImageBedDialsForTesting(t *testing.T) {
	t.Helper()
	previous := imageBedDialContext
	imageBedDialContext = (&net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext
	t.Cleanup(func() {
		imageBedDialContext = previous
	})
}

// TestIsBlockedIP is the core regression for P2-#7. Each entry encodes a
// concrete attack vector a misconfigured image bed could point at.
func TestIsBlockedIP(t *testing.T) {
	cases := []struct {
		name    string
		ip      string
		blocked bool
	}{
		// Blocked
		{"ipv4 loopback", "127.0.0.1", true},
		{"ipv6 loopback", "::1", true},
		{"rfc1918 10", "10.0.0.1", true},
		{"rfc1918 172", "172.16.0.1", true},
		{"rfc1918 192", "192.168.1.1", true},
		{"ula fc00", "fc00::1", true},
		{"cloud metadata aws", "169.254.169.254", true}, // THE big one.
		{"ipv6 link local", "fe80::1", true},
		{"unspecified v4", "0.0.0.0", true},
		{"unspecified v6", "::", true},
		{"multicast v4", "224.0.0.1", true},
		{"multicast v6", "ff02::1", true},
		{"nil ip", "", true},

		// Allowed (public addresses)
		{"public dns cloudflare", "1.1.1.1", false},
		{"public dns google", "8.8.8.8", false},
		{"github", "140.82.112.4", false},
		{"public ipv6 google dns", "2001:4860:4860::8888", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var ip net.IP
			if tc.ip != "" {
				ip = net.ParseIP(tc.ip)
				if ip == nil {
					t.Fatalf("invalid test IP %q", tc.ip)
				}
			}
			got := isBlockedIP(ip)
			if got != tc.blocked {
				t.Fatalf("isBlockedIP(%q) = %v, want %v", tc.ip, got, tc.blocked)
			}
		})
	}
}

// TestSafeDialContext_RejectsLiteralBlockedIP exercises the IP-literal fast
// path. We intentionally dial a port that nothing listens on; the function
// must refuse *before* attempting the connect, so the error is the SSRF
// sentinel, not a connection-refused message.
func TestSafeDialContext_RejectsLiteralBlockedIP(t *testing.T) {
	targets := []string{
		"127.0.0.1:1",         // loopback
		"169.254.169.254:80",  // cloud metadata
		"192.168.0.1:80",      // RFC1918
		"[::1]:80",            // loopback v6
		"[fe80::1]:80",        // link-local v6
	}
	for _, addr := range targets {
		t.Run(addr, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			defer cancel()
			_, err := safeDialContext(ctx, "tcp", addr)
			if err == nil {
				t.Fatalf("expected safeDialContext to refuse %s", addr)
			}
			if !errors.Is(err, errSSRFBlocked) {
				t.Fatalf("expected errSSRFBlocked for %s, got %v", addr, err)
			}
		})
	}
}
