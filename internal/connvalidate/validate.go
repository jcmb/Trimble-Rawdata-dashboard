// Package connvalidate checks receiver targets for browser-initiated connections.
package connvalidate

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// BuildTCPURI returns a normalized tcp:// URI from host and port.
func BuildTCPURI(host string, port int) (string, error) {
	host = strings.TrimSpace(host)
	if host == "" {
		return "", fmt.Errorf("host is required")
	}
	if port < 1 || port > 65535 {
		return "", fmt.Errorf("port must be 1–65535")
	}
	// Preserve IPv6 literals for net.JoinHostPort.
	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		host = host[1 : len(host)-1]
	}
	return "tcp://" + net.JoinHostPort(host, strconv.Itoa(port)), nil
}

// NormalizeTCPURI accepts tcp://host:port or host:port and returns tcp://host:port.
func NormalizeTCPURI(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("connection URI is required")
	}
	if !strings.Contains(raw, "://") {
		raw = "tcp://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse URI: %w", err)
	}
	switch u.Scheme {
	case "tcp", "tcp4", "tcp6":
	default:
		return "", fmt.Errorf("only tcp:// connections are supported from the web UI")
	}
	host := u.Hostname()
	if host == "" {
		return "", fmt.Errorf("host is required")
	}
	portStr := u.Port()
	if portStr == "" {
		return "", fmt.Errorf("port is required")
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return "", fmt.Errorf("invalid port: %w", err)
	}
	return BuildTCPURI(host, port)
}

// HostIsNonPublic reports whether host resolves to loopback, private, or link-local addresses.
func HostIsNonPublic(host string) (bool, error) {
	host = strings.TrimSpace(host)
	host = strings.Trim(host, "[]")
	if host == "" {
		return false, fmt.Errorf("host is required")
	}

	if ip := net.ParseIP(host); ip != nil {
		return isNonPublicIP(ip), nil
	}

	h := strings.ToLower(strings.TrimSuffix(host, "."))
	if h == "localhost" || strings.HasSuffix(h, ".localhost") {
		return true, nil
	}

	ips, err := net.LookupIP(host)
	if err != nil {
		return false, fmt.Errorf("resolve host: %w", err)
	}
	if len(ips) == 0 {
		return false, fmt.Errorf("resolve host: no addresses")
	}
	for _, ip := range ips {
		if isNonPublicIP(ip) {
			return true, nil
		}
	}
	return false, nil
}

func isNonPublicIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	// Unique local (fc00::/7).
	if ip.To4() == nil && len(ip) == net.IPv6len && ip[0] == 0xfc {
		return true
	}
	if ip.To4() == nil && len(ip) == net.IPv6len && ip[0] == 0xfd {
		return true
	}
	return false
}

// ValidateBrowserTCP rejects private/loopback targets unless allowLocal is true.
func ValidateBrowserTCP(uri string, allowLocal bool) (string, error) {
	norm, err := NormalizeTCPURI(uri)
	if err != nil {
		return "", err
	}
	if allowLocal {
		return norm, nil
	}
	u, _ := url.Parse(norm)
	nonPublic, err := HostIsNonPublic(u.Hostname())
	if err != nil {
		return "", err
	}
	if nonPublic {
		return "", fmt.Errorf("connections to local or private addresses are disabled (start server with -allow-local-hosts)")
	}
	return norm, nil
}
