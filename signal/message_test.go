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
	"strings"
	"testing"
	"time"
)

func TestParseMessageJSON(t *testing.T) {
	// One synthetic message covers the JSON sub-parsers that populate mentions,
	// quote, reactions, edit history, and attachment metadata.
	alice := contact("Alice")
	bob := contact("Bob")
	ctx := testContext(map[string]*Recipient{
		"aci-alice": alice,
		"aci-bob":   bob,
	}, map[string]*Recipient{
		"+15551234567": alice,
	}, map[string]*Recipient{})
	ctx.dbVersion = 20

	msg := Message{
		ID:       "message-id",
		TimeSent: 123,
		TimeRecv: 456,
		Body:     MessageBody{Text: "hello \ufffc"},
		JSON: `{
			"attachments": [
				{
					"contentType": "image/jpeg",
					"fileName": "photo.jpg",
					"path": "photo",
					"size": 789
				}
			],
			"bodyRanges": [
				{"start": 6, "length": 1, "mentionAci": "aci-alice"}
			],
			"quote": {
				"id": "99",
				"authorAci": "aci-bob",
				"text": "quoted"
			},
			"reactions": [
				{
					"emoji": "👍",
					"fromId": "+15551234567",
					"targetTimestamp": 123,
					"timestamp": 456
				}
			],
			"editHistory": [
				{
					"body": "edited",
					"timestamp": 777,
					"bodyRanges": [
						{"start": 0, "length": 0, "mentionAci": "aci-bob"}
					]
				}
			]
		}`,
	}

	jmsg, err := ctx.parseMessageJSON(&msg)
	if err != nil {
		t.Fatal(err)
	}
	if len(jmsg.Attachments) != 1 || jmsg.Attachments[0].FileName != "photo.jpg" {
		t.Fatalf("messageJSON attachments: have %+v", jmsg.Attachments)
	}
	if len(msg.Body.Mentions) != 1 || msg.Body.Mentions[0].Recipient != alice {
		t.Fatalf("body mentions: have %+v", msg.Body.Mentions)
	}
	if msg.Quote == nil || msg.Quote.Recipient != bob || msg.Quote.TimeSent != 99 {
		t.Fatalf("quote: have %+v", msg.Quote)
	}
	if len(msg.Reactions) != 1 || msg.Reactions[0].Recipient != alice {
		t.Fatalf("reactions: have %+v", msg.Reactions)
	}
	if len(msg.Edits) != 1 || msg.Edits[0].Body.Text != "edited" || msg.Edits[0].TimeEdit != 777 {
		t.Fatalf("edits: have %+v", msg.Edits)
	}
}

func TestParseMessageJSONInvalid(t *testing.T) {
	// Bad JSON should fail before any partial message fields are trusted.
	ctx := testContext(nil, nil, nil)
	msg := Message{JSON: `{`}
	if _, err := ctx.parseMessageJSON(&msg); err == nil {
		t.Fatal("parseMessageJSON() invalid JSON: no error")
	}
}

func TestErrMentionError(t *testing.T) {
	// Mention errors include UTF-16 positions and placeholders, which are the
	// key details needed to debug Signal body ranges.
	body := MessageBody{
		Text: "a\ufffcb",
		Mentions: []Mention{
			{Start: 1, Length: 1},
		},
	}
	err := (&ErrMention{
		Msg:   "invalid mention",
		Index: 0,
		Body:  &body,
	}).Error()

	for _, want := range []string{
		"invalid mention",
		"index: 0",
		"placeholders: 1",
		"mentions: 0:1,1",
	} {
		if !strings.Contains(err, want) {
			t.Fatalf("ErrMention.Error(): missing %q in %q", want, err)
		}
	}
}

func TestIsOutgoing(t *testing.T) {
	// Several formatters branch only on the literal outgoing message type.
	if !(&Message{Type: "outgoing"}).IsOutgoing() {
		t.Fatal("IsOutgoing() outgoing: want true")
	}
	if (&Message{Type: "incoming"}).IsOutgoing() {
		t.Fatal("IsOutgoing() incoming: want false")
	}
}

func TestConversationMessages(t *testing.T) {
	// A tiny modern schema lets the query-selection paths run without a real
	// Signal Desktop database.
	ctx := messageQueryContext(t)
	conv := Conversation{ID: "conv"}

	tests := []struct {
		name string
		ival Interval
		want []string
	}{
		{
			name: "all",
			want: []string{"m1", "m2", "m3"},
		},
		{
			name: "sent before",
			ival: Interval{Max: time.UnixMilli(2000)},
			want: []string{"m1", "m2"},
		},
		{
			name: "sent after",
			ival: Interval{Min: time.UnixMilli(2000)},
			want: []string{"m2", "m3"},
		},
		{
			name: "sent between",
			ival: Interval{
				Min: time.UnixMilli(1500),
				Max: time.UnixMilli(2500),
			},
			want: []string{"m2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msgs, err := ctx.ConversationMessages(&conv, tt.ival)
			if err != nil {
				t.Fatal(err)
			}
			if len(msgs) != len(tt.want) {
				t.Fatalf("ConversationMessages() len: want %d, have %d", len(tt.want), len(msgs))
			}
			for i, want := range tt.want {
				if msgs[i].ID != want {
					t.Fatalf("ConversationMessages()[%d]: want %q, have %q", i, want, msgs[i].ID)
				}
				if msgs[i].Conversation == nil || msgs[i].Source == nil {
					t.Fatalf("ConversationMessages()[%d] recipients: have conversation=%+v source=%+v", i, msgs[i].Conversation, msgs[i].Source)
				}
			}
		})
	}
}

func TestConversationAttachments(t *testing.T) {
	// ConversationAttachments flattens per-message attachment metadata.
	ctx := messageQueryContext(t)
	atts, err := ctx.ConversationAttachments(&Conversation{ID: "conv"}, Interval{})
	if err != nil {
		t.Fatal(err)
	}
	if len(atts) != 1 {
		t.Fatalf("ConversationAttachments() len: want 1, have %d", len(atts))
	}
	if atts[0].FileName != "photo.jpg" || atts[0].TimeSent != 2000 || atts[0].TimeRecv != 2010 {
		t.Fatalf("ConversationAttachments(): have %+v", atts[0])
	}
}

func messageQueryContext(t *testing.T) Context {
	t.Helper()
	db := memoryDB(t)
	t.Cleanup(func() { db.Close() })

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
		CREATE TABLE messages (
			id TEXT,
			conversationId TEXT,
			sourceServiceId TEXT,
			type TEXT,
			body TEXT,
			json TEXT,
			sent_at INTEGER,
			received_at INTEGER
		);
		INSERT INTO conversations VALUES
			('conv', '{}', 'private', 'Conversation', '', '', '', '+15550000000', 'aci-conv', NULL),
			('source', '{}', 'private', 'Source', '', '', '', '+15551111111', 'aci-source', NULL);
		INSERT INTO messages VALUES
			('m1', 'conv', 'aci-source', 'incoming', 'one', '{"received_at_ms":1010}', 1000, 1010),
			('m2', 'conv', 'aci-source', 'incoming', 'two', '{"received_at_ms":2010,"attachments":[{"contentType":"image/jpeg","fileName":"photo.jpg","path":"photo","size":123}]}', 2000, 2010),
			('m3', 'conv', 'aci-source', 'outgoing', 'three', '{"received_at_ms":3010}', 3000, 3010)
	`); err != nil {
		t.Fatal(err)
	}

	return Context{
		db:        db,
		dbVersion: 88,
	}
}
