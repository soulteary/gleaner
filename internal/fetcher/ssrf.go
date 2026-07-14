package fetcher

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

// ErrBlockedURL is returned when a URL targets a private or otherwise blocked host.
var ErrBlockedURL = errors.New("url points to a blocked host")

var blockedHostnames = map[string]struct{}{
	"localhost":                {},
	"metadata.google.internal": {},
}

func validateFetchURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	switch u.Scheme {
	case "http", "https":
	default:
		return fmt.Errorf("unsupported URL scheme %q", u.Scheme)
	}
	host := u.Hostname()
	if host == "" {
		return errors.New("missing host")
	}
	if isBlockedHostname(host) {
		return fmt.Errorf("%w: %s", ErrBlockedURL, host)
	}
	if ip := net.ParseIP(host); ip != nil && isBlockedIP(ip) {
		return fmt.Errorf("%w: %s", ErrBlockedURL, ip)
	}
	return nil
}

func isBlockedHostname(host string) bool {
	h := strings.ToLower(strings.TrimSuffix(host, "."))
	if _, ok := blockedHostnames[h]; ok {
		return true
	}
	if strings.HasSuffix(h, ".localhost") || strings.HasSuffix(h, ".local") {
		return true
	}
	return false
}

func isBlockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	if ip4 := ip.To4(); ip4 == nil && len(ip) == net.IPv6len {
		// Unique local addresses fc00::/7
		return ip[0]&0xfe == 0xfc
	}
	return false
}

func checkDialAddr(ctx context.Context, addr string) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	if isBlockedHostname(host) {
		return fmt.Errorf("%w: %s", ErrBlockedURL, host)
	}
	if ip := net.ParseIP(host); ip != nil {
		if isBlockedIP(ip) {
			return fmt.Errorf("%w: %s", ErrBlockedURL, ip)
		}
		return nil
	}
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return err
	}
	if len(ips) == 0 {
		return fmt.Errorf("no addresses for host %q", host)
	}
	for _, rec := range ips {
		if isBlockedIP(rec.IP) {
			return fmt.Errorf("%w: %s resolves to %s", ErrBlockedURL, host, rec.IP)
		}
	}
	return nil
}
