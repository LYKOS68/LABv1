package handlers

import (
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// ProxyPolicy vertraut nur ausdrücklich konfigurierten Netzen, nie pauschal privaten IPs.
type ProxyPolicy struct {
	trusted []netip.Prefix
}

func NewProxyPolicy(cidrs string) (*ProxyPolicy, error) {
	p := &ProxyPolicy{}
	if strings.TrimSpace(cidrs) == "" {
		return p, nil
	}
	for _, value := range strings.Split(cidrs, ",") {
		prefix, err := netip.ParsePrefix(strings.TrimSpace(value))
		if err != nil {
			return nil, fmt.Errorf("trusted proxy CIDR: %w", err)
		}
		if prefix.Bits() == 0 || prefix.Addr().Is4In6() {
			return nil, fmt.Errorf("trusted proxy CIDR must be a specific native IPv4 or IPv6 network")
		}
		p.trusted = append(p.trusted, prefix.Masked())
	}
	return p, nil
}

func (p *ProxyPolicy) clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	peer, err := netip.ParseAddr(host)
	if err != nil || peer.Zone() != "" {
		return "unknown"
	}
	peer = peer.Unmap()
	if !p.contains(peer) {
		return peer.String()
	}
	values := r.Header.Values("X-Forwarded-For")
	chain := strings.Join(values, ",")
	if len(chain) == 0 || len(chain) > 4096 {
		return peer.String()
	}
	parts := strings.Split(chain, ",")
	if len(parts) > 32 {
		return peer.String()
	}
	ips := make([]netip.Addr, len(parts))
	for i, part := range parts {
		ip, err := netip.ParseAddr(strings.TrimSpace(part))
		if err != nil || ip.Zone() != "" {
			return peer.String()
		}
		ips[i] = ip.Unmap()
	}
	// Von rechts endet das Vertrauen am ersten unbekannten Hop; linke Angaben können gefälscht sein.
	for i := len(ips) - 1; i >= 0; i-- {
		peer = ips[i]
		if !p.contains(peer) {
			return peer.String()
		}
	}
	return peer.String()
}

func (p *ProxyPolicy) contains(ip netip.Addr) bool {
	if p == nil {
		return false
	}
	for _, prefix := range p.trusted {
		if prefix.Contains(ip) {
			return true
		}
	}
	return false
}

func (h *Handlers) sessionCookieSecure(r *http.Request) bool {
	// Online unabhängig von TLS-Terminierung und manipulierbaren Headern sicher.
	return !h.AllowInsecureLocalHTTP || r.TLS != nil
}
