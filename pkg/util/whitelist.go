package util

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"strings"

	"github.com/grepplabs/reverse-http/pkg/gost"
	"github.com/grepplabs/reverse-http/pkg/logger"
)

type Entry[T any] struct {
	Value   T
	MinPort int
	MaxPort int
}

func (e Entry[T]) String() string {
	if e.MinPort == 0 && e.MaxPort == 0 {
		return fmt.Sprintf("%v", e.Value)
	} else {
		return fmt.Sprintf("%v (%d,%d)", e.Value, e.MinPort, e.MaxPort)
	}
}

func (e Entry[T]) IsPortAllowed(port int) bool {
	if e.MinPort == 0 && e.MaxPort == 0 {
		return true
	}
	return e.MinPort <= port && port <= e.MaxPort
}

type Whitelist struct {
	Networks []Entry[*net.IPNet]
	IPs      []Entry[net.IP]
	Zones    []Entry[string]
	Hosts    []Entry[string]
	logger   *logger.Logger
}

func NewWhitelist() *Whitelist {
	return &Whitelist{
		logger: logger.GetInstance().WithFields(map[string]any{"kind": "whitelist"}),
	}
}

// WhitelistFromStrings creates a Whitelist if the host list is not empty, nil otherwise.
func WhitelistFromStrings(strings []string) *Whitelist {
	if len(strings) == 0 {
		return nil
	}
	wl := NewWhitelist()
	for _, s := range strings {
		wl.AddFromString(s)
	}
	return wl
}

// Contains returns bypass result, `true` blocks the request, `false` pass the request through.
func (p *Whitelist) Contains(ctx context.Context, network, addr string, opts ...gost.BypassOption) bool {
	// localhost
	// localhost:80
	// localhost:1000-2000
	// *.zone
	// *.zone:80
	// *.zone:1000-2000
	// 127.0.0.1
	// 127.0.0.1:80
	// 127.0.0.1:1000-2000
	// 10.0.0.1/8
	// 10.0.0.1/8:80
	// 10.0.0.1/8:1000-2000
	// 1000::/16
	// 1000::/16:80
	// 1000::/16:1000-2000
	// [2001:db8::1]/64
	// [2001:db8::1]/64:80
	// [2001:db8::1]/64:1000-2000
	// 2001:db8::1
	// [2001:db8::1]
	// [2001:db8::1]:80
	// [2001:db8::1]:1000-2000
	host, destPort, err := net.SplitHostPort(addr)
	if err != nil {
		p.logger.Error("blocked", slog.String("error", err.Error()))
		return true // block
	}
	port, err := strconv.Atoi(destPort)
	if err != nil {
		port = -1
	}
	blocked := !p.IsAddrAllowed(host, port)
	if blocked {
		p.logger.Infof("blocked %s", addr)
	}
	return blocked
}

func (p *Whitelist) IsPortAllowed(port int, entry *Entry[any]) bool {
	if entry.MinPort == 0 && entry.MaxPort == 0 {
		return true
	}
	return entry.MinPort <= port && port <= entry.MaxPort
}

func (p *Whitelist) IsAddrAllowed(host string, port int) bool {
	if ip := net.ParseIP(host); ip != nil {
		for _, ipNet := range p.Networks {
			if ipNet.Value.Contains(ip) {
				// return true
				return ipNet.IsPortAllowed(port)
			}
		}
		for _, bypassIP := range p.IPs {
			if bypassIP.Value.Equal(ip) {
				// return true
				return bypassIP.IsPortAllowed(port)
			}
		}
		return false
	}

	for _, zone := range p.Zones {
		if strings.HasSuffix(host, zone.Value) {
			// return true
			return zone.IsPortAllowed(port)
		}
		if host == zone.Value[1:] {
			// For a zone ".example.com", we match "example.com" too.
			// return true
			return zone.IsPortAllowed(port)
		}
	}
	for _, bypassHost := range p.Hosts {
		if bypassHost.Value == host {
			//return true
			return bypassHost.IsPortAllowed(port)
		}
	}
	return false
}

func (p *Whitelist) AddFromString(s string) {
	entries := strings.Split(s, ",")
	for _, entry := range entries {
		host, minPort, maxPort, err := toAddrPorts(entry)
		if err != nil {
			continue
		}
		host = strings.TrimSpace(host)
		if len(host) == 0 {
			continue
		}
		if strings.Contains(host, "/") {
			// We assume that it's a CIDR address like 127.0.0.0/8
			if _, ipNet, err := net.ParseCIDR(host); err == nil {
				p.AddNetwork(ipNet, minPort, maxPort)
			}
			continue
		}
		if ip := net.ParseIP(host); ip != nil {
			p.AddIP(ip, minPort, maxPort)
			continue
		}
		if strings.HasPrefix(host, "*.") {
			p.AddZone(host[1:], minPort, maxPort)
			continue
		}
		p.AddHost(host, minPort, maxPort)
	}
}

// AddIP specifies an IP address that will use the bypass proxy. Note that
// this will only take effect if a literal IP address is dialed. A connection
// to a named host will never match an IP.
func (p *Whitelist) AddIP(ip net.IP, minPort, maxPort int) {
	p.IPs = append(p.IPs, Entry[net.IP]{
		Value:   ip,
		MinPort: minPort,
		MaxPort: maxPort,
	})
}

// AddNetwork specifies an IP range that will use the bypass proxy. Note that
// this will only take effect if a literal IP address is dialed. A connection
// to a named host will never match.
func (p *Whitelist) AddNetwork(ipNet *net.IPNet, minPort, maxPort int) {
	p.Networks = append(p.Networks, Entry[*net.IPNet]{
		Value:   ipNet,
		MinPort: minPort,
		MaxPort: maxPort,
	})
}

// AddZone specifies a DNS suffix that will use the bypass proxy. A zone of
// "example.com" matches "example.com" and all of its subdomains.
func (p *Whitelist) AddZone(zone string, minPort, maxPort int) {
	zone = strings.TrimSuffix(zone, ".")
	if !strings.HasPrefix(zone, ".") {
		zone = "." + zone
	}
	p.Zones = append(p.Zones, Entry[string]{
		Value:   zone,
		MinPort: minPort,
		MaxPort: maxPort,
	})
}

// AddHost specifies a host name that will use the bypass proxy.
func (p *Whitelist) AddHost(host string, minPort, maxPort int) {
	host = strings.TrimSuffix(host, ".")
	p.Hosts = append(p.Hosts, Entry[string]{
		Value:   host,
		MinPort: minPort,
		MaxPort: maxPort,
	})
}

func toAddrPorts(input string) (string, int, int, error) {
	addr, ports := parseAddrPorts(input)
	addr = strings.TrimPrefix(addr, "[")
	addr = strings.Replace(addr, "]", "", 1)
	if addr == "" {
		return "", 0, 0, fmt.Errorf("invalid address: input %s", input)
	}
	if ports != "" {
		parts := strings.Split(ports, "-")
		switch len(parts) {
		case 1:
			port, err := strconv.Atoi(parts[0])
			if err != nil {
				return "", 0, 0, fmt.Errorf("invalid port: input %s: %w", input, err)
			}
			return addr, port, port, nil
		case 2:
			port1, err := strconv.Atoi(parts[0])
			if err != nil {
				return "", 0, 0, fmt.Errorf("invalid port: input %s: %w", input, err)
			}
			port2, err := strconv.Atoi(parts[1])
			if err != nil {
				return "", 0, 0, fmt.Errorf("invalid port: input %s: %w", input, err)
			}
			return addr, port1, port2, nil
		default:
			return "", 0, 0, fmt.Errorf("invalid ports: input %s", input)
		}
	}
	return addr, 0, 0, nil
}

func parseAddrPorts(input string) (string, string) {
	if strings.HasPrefix(input, "[") {
		idx := strings.Index(input, "]")
		if idx == -1 {
			return "", ""
		}
		if len(input) > idx+2 {
			idx2 := strings.LastIndex(input, ":")
			if idx2 > idx {
				return input[:idx2], input[idx2+1:]
			} else {
				return input, ""
			}
		}
		return input[:idx+1], ""
	} else {
		idx := strings.Index(input, "/")
		if idx != -1 {
			idx2 := strings.LastIndex(input, ":")
			if idx2 > idx {
				return input[:idx2], input[idx2+1:]
			} else {
				return input, ""
			}
		} else {
			ip := net.ParseIP(input)
			// ipv6
			if ip != nil && strings.Contains(input, ":") {
				return ip.String(), ""
			} else {
				idx2 := strings.LastIndex(input, ":")
				if idx2 != -1 {
					return input[:idx2], input[idx2+1:]
				} else {
					return input, ""
				}
			}
		}
	}
}
