package application

import "testing"

func TestAllowsIP(t *testing.T) {
	tests := []struct {
		name  string
		cidrs []string
		ip    string
		want  bool
	}{
		{name: "empty allowlist permits", ip: "203.0.113.4", want: true},
		{name: "ipv4 included", cidrs: []string{"10.20.0.0/16"}, ip: "10.20.4.9", want: true},
		{name: "ipv4 excluded", cidrs: []string{"10.20.0.0/16"}, ip: "10.21.4.9"},
		{name: "ipv6 included", cidrs: []string{"2001:db8::/32"}, ip: "2001:db8::8", want: true},
		{name: "malformed remote rejected", cidrs: []string{"10.20.0.0/16"}, ip: "not-an-ip"},
		{name: "malformed stored prefix ignored", cidrs: []string{"broken"}, ip: "10.20.4.9"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := allowsIP(tt.cidrs, tt.ip); got != tt.want {
				t.Fatalf("allowsIP() = %v, want %v", got, tt.want)
			}
		})
	}
}
