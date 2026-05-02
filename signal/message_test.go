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
