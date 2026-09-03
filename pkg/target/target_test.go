package target

import (
	"context"
	"net"
	"testing"
)

func TestIsPublicIP(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want bool
	}{
		{name: "public IPv4", ip: "8.8.8.8", want: true},
		{name: "public IPv6", ip: "2001:4860:4860::8888", want: true},
		{name: "private IPv4", ip: "10.0.0.1", want: false},
		{name: "shared IPv4", ip: "100.64.0.1", want: false},
		{name: "loopback", ip: "127.0.0.1", want: false},
		{name: "link local", ip: "169.254.1.1", want: false},
		{name: "documentation", ip: "192.0.2.1", want: false},
		{name: "benchmark", ip: "198.18.0.1", want: false},
		{name: "private IPv6", ip: "fd00::1", want: false},
		{name: "IPv6 documentation", ip: "2001:db8::1", want: false},
		{name: "multicast IPv6", ip: "ff02::1", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsPublicIP(net.ParseIP(tt.ip)); got != tt.want {
				t.Fatalf("IsPublicIP(%q) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}

func TestResolvePublicRejectsPrivateIPLiteral(t *testing.T) {
	for _, host := range []string{"10.0.0.1", "127.0.0.1", "192.168.1.1", "::1", "fd00::1"} {
		if _, err := ResolvePublic(context.Background(), host); err != ErrNoPublicAddress {
			t.Errorf("ResolvePublic(%q) error = %v, want %v", host, err, ErrNoPublicAddress)
		}
	}
}

func TestResolvePublicAcceptsPublicIPLiteral(t *testing.T) {
	addresses, err := ResolvePublic(context.Background(), "8.8.8.8")
	if err != nil {
		t.Fatalf("ResolvePublic(public IP) returned error: %v", err)
	}
	if len(addresses) != 1 || !addresses[0].Equal(net.ParseIP("8.8.8.8")) {
		t.Fatalf("ResolvePublic(public IP) = %v, want [8.8.8.8]", addresses)
	}
}
