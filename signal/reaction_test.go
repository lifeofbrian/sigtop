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

import "testing"

func TestParseReactionJSON(t *testing.T) {
	// Reaction JSON carries both the emoji and the sender ID that must be
	// resolved through version-dependent recipient lookup.
	alice := contact("Alice")
	ctx := testContext(nil, map[string]*Recipient{
		"+15551234567": alice,
	}, map[string]*Recipient{})
	ctx.dbVersion = 20

	msg := Message{}
	err := ctx.parseReactionJSON(&msg, &messageJSON{
		Reactions: []reactionJSON{
			{
				Emoji:           "👍",
				FromID:          "+15551234567",
				TargetTimestamp: 123,
				Timestamp:       456,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(msg.Reactions) != 1 {
		t.Fatalf("parseReactionJSON() len: want 1, have %d", len(msg.Reactions))
	}
	rct := msg.Reactions[0]
	if rct.Recipient != alice || rct.Emoji != "👍" || rct.TimeSent != 123 || rct.TimeRecv != 456 {
		t.Fatalf("parseReactionJSON(): have %+v", rct)
	}
}

func TestRecipientFromReactionID(t *testing.T) {
	// Database version 20 changed reaction sender IDs from conversation IDs to
	// phone-number-style IDs for contacts.
	alice := contact("Alice")
	bob := contact("Bob")

	oldCtx := testContext(nil, nil, map[string]*Recipient{
		"15551234567": alice,
	})
	oldCtx.dbVersion = 19
	if got, err := oldCtx.recipientFromReactionID("+15551234567"); err != nil || got != alice {
		t.Fatalf("recipientFromReactionID() old: want Alice nil, have %+v %v", got, err)
	}

	newCtx := testContext(nil, map[string]*Recipient{
		"+15551234567": alice,
	}, map[string]*Recipient{
		"conversation-bob": bob,
	})
	newCtx.dbVersion = 20
	if got, err := newCtx.recipientFromReactionID("+15551234567"); err != nil || got != alice {
		t.Fatalf("recipientFromReactionID() phone: want Alice nil, have %+v %v", got, err)
	}
	if got, err := newCtx.recipientFromReactionID("conversation-bob"); err != nil || got != bob {
		t.Fatalf("recipientFromReactionID() conversation: want Bob nil, have %+v %v", got, err)
	}
}

func testContext(byACI, byPhone, byConversation map[string]*Recipient) Context {
	// Seed recipient maps directly so parser tests avoid database setup.
	if byACI == nil {
		byACI = make(map[string]*Recipient)
	}
	if byPhone == nil {
		byPhone = make(map[string]*Recipient)
	}
	if byConversation == nil {
		byConversation = make(map[string]*Recipient)
	}
	return Context{
		recipientsByACI:            byACI,
		recipientsByPhone:          byPhone,
		recipientsByConversationID: byConversation,
	}
}
