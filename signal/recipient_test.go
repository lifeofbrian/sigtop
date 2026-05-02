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
	"os"
	"path/filepath"
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

func TestMakeRecipientMapsAndConversations(t *testing.T) {
	// A tiny conversations table exercises recipient JSON parsing, lookup maps,
	// bidi trimming, and the built-in Signal avatar suppression.
	db := memoryDB(t)
	defer db.Close()
	if err := db.Exec(`
		CREATE TABLE conversations (
			id TEXT,
			json TEXT,
			type TEXT,
			name TEXT,
			profileName TEXT,
			profileFamilyName TEXT,
			profileFullName TEXT,
			e164 TEXT,
			serviceId TEXT,
			groupId TEXT
		)
	`); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`
		INSERT INTO conversations VALUES (
			'contact-id',
			'{"username":"alice.123","profileAvatar":{"path":"images/profile-avatar.svg"}}',
			'private',
			'⁨Alice⁩',
			'Alice',
			'Family',
			'Alice Family',
			'+15551234567',
			'ACI-ALICE',
			NULL
		);
		INSERT INTO conversations VALUES (
			'group-conversation',
			'{}',
			'group',
			'Study Group',
			NULL,
			NULL,
			NULL,
			NULL,
			NULL,
			'group-id'
		)
	`); err != nil {
		t.Fatal(err)
	}

	ctx := Context{
		db:        db,
		dbVersion: 88,
	}
	convs, err := ctx.Conversations()
	if err != nil {
		t.Fatal(err)
	}
	if len(convs) != 2 {
		t.Fatalf("Conversations() len: want 2, have %d", len(convs))
	}

	contact, err := ctx.recipientFromConversationID("contact-id")
	if err != nil {
		t.Fatal(err)
	}
	if contact.Contact.Name != "Alice" || contact.Contact.Username != "alice.123" {
		t.Fatalf("contact recipient: have %+v", contact.Contact)
	}
	if contact.ProfileAvatar.Path != "" {
		t.Fatalf("Signal release avatar path should be ignored, have %q", contact.ProfileAvatar.Path)
	}
	if got, err := ctx.recipientFromPhone("+15551234567"); err != nil || got != contact {
		t.Fatalf("recipientFromPhone(): want contact nil, have %+v %v", got, err)
	}
	if got, err := ctx.recipientFromACI("aci-alice"); err != nil || got != contact {
		t.Fatalf("recipientFromACI(): want contact nil, have %+v %v", got, err)
	}

	group, err := ctx.recipientFromConversationID("group-conversation")
	if err != nil {
		t.Fatal(err)
	}
	if group.Type != RecipientTypeGroup || group.Group.Name != "Study Group" {
		t.Fatalf("group recipient: have %+v", group)
	}
}

func TestMakeRecipientMapsErrors(t *testing.T) {
	// Unknown recipient types should fail loudly when reading the conversations
	// table.
	db := memoryDB(t)
	defer db.Close()
	if err := db.Exec(`
		CREATE TABLE conversations (
			id TEXT,
			json TEXT,
			type TEXT,
			name TEXT,
			profileName TEXT,
			profileFamilyName TEXT,
			profileFullName TEXT,
			e164 TEXT,
			serviceId TEXT,
			groupId TEXT
		);
		INSERT INTO conversations VALUES ('id', '{}', 'mystery', '', '', '', '', '', '', '')
	`); err != nil {
		t.Fatal(err)
	}

	ctx := Context{
		db:        db,
		dbVersion: 88,
	}
	if err := ctx.makeRecipientMaps(); err == nil {
		t.Fatal("makeRecipientMaps() unknown type: no error")
	}
}

func TestReadAvatar(t *testing.T) {
	// Avatars share the attachment-file reader.
	dir := t.TempDir()
	attDir := filepath.Join(dir, AttachmentDir)
	if err := os.Mkdir(attDir, 0777); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(attDir, "avatar"), []byte("image"), 0666); err != nil {
		t.Fatal(err)
	}

	ctx := Context{dir: dir}
	data, err := ctx.ReadAvatar(&Avatar{
		attachmentFile: attachmentFile{
			Version: 1,
			Path:    "avatar",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "image" {
		t.Fatalf("ReadAvatar(): want image, have %q", data)
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
