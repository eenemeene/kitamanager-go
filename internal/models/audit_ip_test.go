package models

import "testing"

// AnonymizeIP is what stands between an org admin's audit feed and a
// colleague's home address, so its edge cases are worth naming individually.
func TestAnonymizeIP(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"ipv4 keeps the /24", "192.0.2.147", "192.0.2.0"},
		{"ipv4 already on a network boundary", "192.0.2.0", "192.0.2.0"},
		{"ipv4 loopback", "127.0.0.1", "127.0.0.0"},
		// An IPv4-mapped IPv6 address is an IPv4 address wearing a hat.
		// Masking it as a v6 prefix would keep all four octets.
		{"ipv4-mapped ipv6 is treated as ipv4", "::ffff:192.0.2.147", "192.0.2.0"},
		{"ipv6 keeps the /48", "2001:db8:1234:5678:9abc:def0:1234:5678", "2001:db8:1234::"},
		{"ipv6 loopback", "::1", "::"},
		// Empty stays empty: there was nothing recorded to reduce.
		{"empty", "", ""},
		// Fail closed. A value that will not parse cannot be reduced, so it
		// must not be passed through to a viewer who is not entitled to it.
		{"unparseable", "not-an-ip", ""},
		{"partial", "192.0.2", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AnonymizeIP(tt.in); got != tt.want {
				t.Errorf("AnonymizeIP(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// The /48 choice is load-bearing rather than arbitrary: a residential IPv6
// allocation is typically a /56 or /64, so keeping 64 bits would preserve the
// household-level identification the reduction exists to remove.
func TestAnonymizeIP_IPv6DropsTheHouseholdBits(t *testing.T) {
	sameAllocation := []string{
		"2001:db8:1234:5600::1",
		"2001:db8:1234:5601::1",
		"2001:db8:1234:ffff::abcd",
	}
	first := AnonymizeIP(sameAllocation[0])
	for _, ip := range sameAllocation[1:] {
		if got := AnonymizeIP(ip); got != first {
			t.Errorf("addresses inside one /48 must reduce alike: %q gave %q, want %q", ip, got, first)
		}
	}
	if first == "" {
		t.Fatal("expected a non-empty prefix")
	}
}

func TestAuditLogResponse_WithAnonymizedIP(t *testing.T) {
	t.Run("flags the reduction", func(t *testing.T) {
		got := AuditLogResponse{IPAddress: "192.0.2.147"}.WithAnonymizedIP()
		if got.IPAddress != "192.0.2.0" {
			t.Errorf("IPAddress = %q, want %q", got.IPAddress, "192.0.2.0")
		}
		if !got.IPAnonymized {
			t.Error("IPAnonymized must be true so a client can tell a prefix from a real address")
		}
	})

	t.Run("an absent address is not a redaction", func(t *testing.T) {
		got := AuditLogResponse{IPAddress: ""}.WithAnonymizedIP()
		if got.IPAnonymized {
			t.Error("IPAnonymized must stay false when there was no address to reduce")
		}
	})

	t.Run("an unparseable address is still a redaction", func(t *testing.T) {
		// The value is dropped, so reporting it as merely absent would
		// misdescribe what happened to the viewer.
		got := AuditLogResponse{IPAddress: "garbage"}.WithAnonymizedIP()
		if got.IPAddress != "" {
			t.Errorf("IPAddress = %q, want empty", got.IPAddress)
		}
		if !got.IPAnonymized {
			t.Error("IPAnonymized must be true: the recorded value was withheld")
		}
	})

	t.Run("leaves the rest of the row alone", func(t *testing.T) {
		in := AuditLogResponse{IPAddress: "192.0.2.147", UserEmail: "a@example.com", Action: "employee_delete"}
		got := in.WithAnonymizedIP()
		if got.UserEmail != in.UserEmail || got.Action != in.Action {
			t.Error("only the address may change")
		}
	})
}
