package media

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"
)

// imageBedDialContext is the dial function used by imageBedHTTPClient. It is
// a package-level variable so test helpers can swap it for a vanilla dialer
// when exercising httptest servers (which always bind to loopback and would
// otherwise be rejected by safeDialContext). Production always uses
// safeDialContext.
var imageBedDialContext = safeDialContext

// errSSRFBlocked is returned when a DialContext target resolves only to
// addresses that are not allowed to be contacted (loopback, private networks,
// link-local / cloud metadata, unspecified, multicast). It is a sentinel so
// tests and callers can assert on it.
var errSSRFBlocked = errors.New("ssrf: target resolves to a blocked address range")

// isBlockedIP reports whether an IP must not be contacted by user-configured
// image bed uploaders (or any other outbound path that forwards user input).
//
// The checks below map onto the concrete attack vectors most relevant to a
// self-hosted deployment:
//
//   - Loopback (127.0.0.0/8, ::1):          talking to the local machine
//   - Private RFC1918 + ULA (10/8, 172.16/12, 192.168/16, fc00::/7): LAN pivot
//   - Link-local (169.254.0.0/16, fe80::/10): CLOUD METADATA endpoints on AWS,
//                                              GCP, Azure (e.g. 169.254.169.254)
//   - Unspecified (0.0.0.0, ::):            "connect to whatever"
//   - Multicast:                             noise, occasional routing tricks
//
// Net/IP's built-in helpers cover all of the above, so there is no need to
// hand-roll CIDR parsing.
func isBlockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() {
		return true
	}
	if ip.IsPrivate() {
		return true
	}
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	if ip.IsUnspecified() {
		return true
	}
	if ip.IsMulticast() {
		return true
	}
	return false
}

// safeDialContext resolves the target host, drops every blocked IP, and
// dials the first surviving candidate directly by IP. Dialing by IP (rather
// than by hostname) closes the DNS-rebinding window: even if the attacker's
// DNS server returns a safe IP to our LookupIP call and a private IP to a
// later lookup, our subsequent Dial skips name resolution entirely.
func safeDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}

	// Already a literal IP? Validate it directly and skip DNS.
	if ip := net.ParseIP(host); ip != nil {
		if isBlockedIP(ip) {
			return nil, fmt.Errorf("%w: %s", errSSRFBlocked, ip.String())
		}
		dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
		return dialer.DialContext(ctx, network, addr)
	}

	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
	if err != nil {
		return nil, err
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("no addresses resolved for host %s", host)
	}

	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	var lastErr error
	for _, ip := range ips {
		if isBlockedIP(ip) {
			lastErr = fmt.Errorf("%w: %s -> %s", errSSRFBlocked, host, ip.String())
			continue
		}
		conn, dialErr := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		if dialErr == nil {
			return conn, nil
		}
		lastErr = dialErr
	}
	if lastErr == nil {
		lastErr = errSSRFBlocked
	}
	return nil, lastErr
}
