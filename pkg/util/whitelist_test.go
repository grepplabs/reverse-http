package util

import (
	"context"
	"net"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestToAddrPorts(t *testing.T) {
	tests := []struct {
		input   string
		addr    string
		minPort int
		maxPort int
	}{
		{
			input: "localhost",
			addr:  "localhost",
		},
		{
			input:   "localhost:80",
			addr:    "localhost",
			minPort: 80,
			maxPort: 80,
		},
		{
			input:   "localhost:1000-2000",
			addr:    "localhost",
			minPort: 1000,
			maxPort: 2000,
		},
		{
			input: "*.zone",
			addr:  "*.zone",
		},
		{
			input:   "*.zone:80",
			addr:    "*.zone",
			minPort: 80,
			maxPort: 80,
		},
		{
			input:   "*.zone:1000-2000",
			addr:    "*.zone",
			minPort: 1000,
			maxPort: 2000,
		},
		{
			input: "127.0.0.1",
			addr:  "127.0.0.1",
		},
		{
			input:   "127.0.0.1:80",
			addr:    "127.0.0.1",
			minPort: 80,
			maxPort: 80,
		},
		{
			input:   "127.0.0.1:1000-2000",
			addr:    "127.0.0.1",
			minPort: 1000,
			maxPort: 2000,
		},
		{
			input: "10.0.0.1/8",
			addr:  "10.0.0.1/8",
		},
		{
			input:   "10.0.0.1/8:80",
			addr:    "10.0.0.1/8",
			minPort: 80,
			maxPort: 80,
		},
		{
			input:   "10.0.0.1/8:1000-2000",
			addr:    "10.0.0.1/8",
			minPort: 1000,
			maxPort: 2000,
		},
		{
			input: "1000::/16",
			addr:  "1000::/16",
		},
		{
			input:   "1000::/16:80",
			addr:    "1000::/16",
			minPort: 80,
			maxPort: 80,
		},
		{
			input:   "1000::/16:1000-2000",
			addr:    "1000::/16",
			minPort: 1000,
			maxPort: 2000,
		},
		{
			input: "[2001:db8::1]/64",
			addr:  "2001:db8::1/64",
		},
		{
			input:   "[2001:db8::1]/64:80",
			addr:    "2001:db8::1/64",
			minPort: 80,
			maxPort: 80,
		},
		{
			input:   "[2001:db8::1]/64:1000-2000",
			addr:    "2001:db8::1/64",
			minPort: 1000,
			maxPort: 2000,
		},
		{
			input: "2001:db8::1",
			addr:  "2001:db8::1",
		},
		{
			input: "[2001:db8::1]",
			addr:  "2001:db8::1",
		},
		{
			input:   "[2001:db8::1]:80",
			addr:    "2001:db8::1",
			minPort: 80,
			maxPort: 80,
		},
		{
			input:   "[2001:db8::1]:1000-2000",
			addr:    "2001:db8::1",
			minPort: 1000,
			maxPort: 2000,
		},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			addr, minPort, maxPort, err := toAddrPorts(tc.input)
			require.Nil(t, err)
			require.Equal(t, tc.addr, addr)
			require.Equal(t, tc.minPort, minPort)
			require.Equal(t, tc.maxPort, maxPort)
		})
	}
}

func TestWhitelist(t *testing.T) {
	tests := []struct {
		name    string
		list    string
		allowed []string
		blocked []string
	}{

		{
			name: "host while list",
			list: "localhost,*.zone,127.0.0.1,10.0.0.1/8,1000::/16",
			allowed: []string{
				"localhost:123",
				"zone:123",
				"foo.zone:123",
				"127.0.0.1:123",
				"10.1.2.3:123",
				"[1000::]:123",
			},
			blocked: []string{
				"example.com:123",
				"1.2.3.4:123",
				"[1001::]:123",
				"172.217.7.14:443",
				"[2607:f8b0:4006:800::200e]:443",
				"example.com:80",
			},
		},
		{
			name: "all IPv4 networks",
			list: "0.0.0.0/0",
			allowed: []string{
				"0.0.0.0:6443",
				"0.0.0.0:6444",
				"1.2.3.4:123",
				"172.217.7.14:443",
			},
			blocked: []string{
				"example.com:123",
				"[1001::]:123",
				"[2607:f8b0:4006:800::200e]:443",
				"example.com:80",
				"localhost:123",
				"zone:123",
				"foo.zone:123",
			},
		},
		{
			name: "all IPv6 networks",
			list: "0::/0",
			allowed: []string{
				"[1001::]:123",
				"[2607:f8b0:4006:800::200e]:443",
			},
			blocked: []string{
				"0.0.0.0:6443",
				"1.2.3.4:123",
				"172.217.7.14:443",
				"example.com:123",
				"example.com:80",
				"localhost:123",
				"zone:123",
				"foo.zone:123",
			},
		},
		{
			name: "port while list",
			list: "localhost:8080,*.zone:8080,127.0.0.1:8080,10.0.0.1/8:8080,1000::/16:8080",
			allowed: []string{
				"10.1.2.3:8080",
				"127.0.0.1:8080",
				"localhost:8080",
				"zone:8080",
				"foo.zone:8080",
				"[1000::]:8080",
			},
			blocked: []string{
				"10.1.2.3:123",
				"127.0.0.1:123",
				"localhost:123",
				"zone:123",
				"foo.zone:123",
				"[1000::]:123",
				"example.com:123",
				"1.2.3.4:123",
				"[1001::]:123",
				"172.217.7.14:443",
				"[2607:f8b0:4006:800::200e]:443",
				"example.com:80",
			},
		},
		{
			name: "port 8080 on all IPv4 networks",
			list: "0.0.0.0/0:8080",
			allowed: []string{
				"0.0.0.0:8080",
				"1.2.3.4:8080",
				"172.217.7.14:8080",
			},
			blocked: []string{
				"0.0.0.0:6443",
				"0.0.0.0:6444",
				"1.2.3.4:123",
				"172.217.7.14:443",
				"example.com:123",
				"[1001::]:123",
				"[2607:f8b0:4006:800::200e]:443",
				"example.com:80",
				"localhost:123",
				"zone:123",
				"foo.zone:123",
			},
		},
		{
			name: "port range while list",
			list: "localhost:4000-5000,*.zone:4000-5000,127.0.0.1:4000-5000,10.0.0.1/8:4000-5000,1000::/16:4000-5000",
			allowed: []string{

				"10.1.2.3:4000",
				"127.0.0.1:4000",
				"localhost:4000",
				"zone:4000",
				"foo.zone:4000",
				"[1000::]:4000",

				"10.1.2.3:4500",
				"127.0.0.1:4500",
				"localhost:4500",
				"zone:4500",
				"foo.zone:4500",
				"[1000::]:4500",

				"10.1.2.3:5000",
				"127.0.0.1:5000",
				"localhost:5000",
				"zone:5000",
				"foo.zone:5000",
				"[1000::]:5000",
			},
			blocked: []string{
				"10.1.2.3:3999",
				"127.0.0.1:3999",
				"localhost:3999",
				"zone:3999",
				"foo.zone:3999",
				"[1000::]:3999",
				"example.com:3999",
				"1.2.3.4:3999",
				"[1001::]:3999",
				"172.217.7.14:3999",
				"[2607:f8b0:4006:800::200e]:3999",
				"example.com:3999",

				"10.1.2.3:5001",
				"127.0.0.1:5001",
				"localhost:5001",
				"zone:5001",
				"foo.zone:5001",
				"[1000::]:5001",
				"example.com:5001",
				"1.2.3.4:5001",
				"[1001::]:5001",
				"172.217.7.14:5001",
				"[2607:f8b0:4006:800::200e]:5001",
				"example.com:5001",
			},
		},
		{
			name: "port range 4000-5000 on all IPv4 networks",
			list: "0.0.0.0/0:4000-5000",
			allowed: []string{
				"0.0.0.0:4000",
				"1.2.3.4:4000",
				"172.217.7.14:4000",

				"0.0.0.0:4500",
				"1.2.3.4:4500",
				"172.217.7.14:4500",

				"0.0.0.0:5000",
				"1.2.3.4:5000",
				"172.217.7.14:5000",
			},
			blocked: []string{
				"0.0.0.0:6443",
				"0.0.0.0:6444",
				"1.2.3.4:123",
				"172.217.7.14:443",
				"example.com:123",
				"[1001::]:123",
				"[2607:f8b0:4006:800::200e]:443",
				"example.com:80",
				"localhost:123",
				"zone:123",
				"foo.zone:123",

				"0.0.0.0:3999",
				"1.2.3.4:3999",
				"172.217.7.14:3999",

				"0.0.0.0:5001",
				"1.2.3.4:5001",
				"172.217.7.14:5001",
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wl := NewWhitelist()
			wl.AddFromString(tc.list)

			for _, addr := range tc.allowed {
				host, sport, err := net.SplitHostPort(addr)
				require.Nil(t, err)
				port, err := strconv.Atoi(sport)
				require.Nil(t, err)
				allowed := wl.IsAddrAllowed(host, port)
				require.True(t, allowed, "addr %s should be allowed but it is blocked", addr)
				require.False(t, wl.Contains(context.Background(), "tcp", addr), "bypass for %s should not be blocked", addr)

			}
			for _, addr := range tc.blocked {
				host, sport, err := net.SplitHostPort(addr)
				require.Nil(t, err)
				port, err := strconv.Atoi(sport)
				require.Nil(t, err)
				allowed := wl.IsAddrAllowed(host, port)
				require.False(t, allowed, "addr %s should be blocked but it is allowed", addr)
				require.True(t, wl.Contains(context.Background(), "tcp", addr), "bypass for %s should be blocked", addr)
			}
		})
	}
}
