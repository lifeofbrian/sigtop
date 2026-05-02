// Copyright (c) 2026 Tim van der Molen <tim@kariliq.nl>
//
// Permission to use, copy, modify, and distribute this software for any
// purpose with or without fee is hereby granted, provided that the above
// copyright notice and this permission notice appear in all copies.
//
// THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
// WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
// MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
// ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
// WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
// ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
// OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.

package signal

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestTrimBidiChars(t *testing.T) {
	// Signal wraps some contact names in bidi isolate marks; only one matching
	// surrounding pair should be removed.
	tests := []struct {
		in   string
		want string
	}{
		{"\u2068Alice\u2069", "Alice"},
		{"Alice", "Alice"},
		{"\u2068Alice", "\u2068Alice"},
		{"Alice\u2069", "Alice\u2069"},
		{"\u2068Alice\u2069\u2069", "Alice\u2069"},
	}

	for _, tt := range tests {
		if got := trimBidiChars(tt.in); got != tt.want {
			t.Fatalf("trimBidiChars(%q): want %q, have %q", tt.in, tt.want, got)
		}
	}
}

func TestRecipientDisplayNameAndDetailContact(t *testing.T) {
	// Contact display names and details have explicit fallback precedence.
	tests := []struct {
		name string
		rpt  *Recipient
		want string
	}{
		{
			name: "explicit name and phone",
			rpt: contactWith(Contact{
				Name:     "Alice",
				Phone:    "+15551234567",
				Username: "alice.123",
				ACI:      "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
			}),
			want: "Alice (+15551234567)",
		},
		{
			name: "joined profile name and username",
			rpt: contactWith(Contact{
				ProfileName:       "Alice",
				ProfileJoinedName: "Alice Baker",
				Username:          "alice.123",
				ACI:               "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
			}),
			want: "Alice Baker (alice.123)",
		},
		{
			name: "profile name and ACI",
			rpt: contactWith(Contact{
				ProfileName: "Alice",
				ACI:         "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
			}),
			want: "Alice (aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee)",
		},
		{
			name: "unknown contact with no detail",
			rpt:  contactWith(Contact{}),
			want: "Unknown",
		},
		{
			name: "nil recipient",
			rpt:  nil,
			want: "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rpt.DetailedDisplayName(); got != tt.want {
				t.Fatalf("DetailedDisplayName(): want %q, have %q", tt.want, got)
			}
		})
	}
}

func TestRecipientDisplayNameAndDetailGroup(t *testing.T) {
	// New group IDs are base64url-like; older raw byte IDs are rendered as hex.
	rawID := strings.Repeat("\xff", 32)
	encodedID := base64.StdEncoding.EncodeToString([]byte(rawID))

	tests := []struct {
		name string
		rpt  *Recipient
		want string
	}{
		{
			name: "new group ID is base64url without padding",
			rpt: groupWith(Group{
				Name: "Family",
				ID:   encodedID,
			}),
			want: "Family (" + strings.Repeat("_", 42) + "8)",
		},
		{
			name: "old group ID is hex encoded",
			rpt: groupWith(Group{
				Name: "Family",
				ID:   "abc",
			}),
			want: "Family (616263)",
		},
		{
			name: "unknown group keeps hex detail",
			rpt: groupWith(Group{
				ID: "abc",
			}),
			want: "Unknown (616263)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rpt.DetailedDisplayName(); got != tt.want {
				t.Fatalf("DetailedDisplayName(): want %q, have %q", tt.want, got)
			}
		})
	}
}

func contactWith(c Contact) *Recipient {
	return &Recipient{
		Type:    RecipientTypeContact,
		Contact: c,
	}
}

func groupWith(g Group) *Recipient {
	return &Recipient{
		Type:  RecipientTypeGroup,
		Group: g,
	}
}
