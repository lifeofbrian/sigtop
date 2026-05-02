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

package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/tbvdm/sigtop/errio"
	"github.com/tbvdm/sigtop/signal"
)

func TestJSONWriteMessages(t *testing.T) {
	// Preserve the export format as a JSON array of raw message JSON objects.
	var buf bytes.Buffer
	ew := errio.NewWriter(&buf)
	msgs := []signal.Message{
		{JSON: `{"id":1}`},
		{JSON: `{"id":2}`},
	}

	if err := jsonWriteMessages(ew, msgs); err != nil {
		t.Fatal(err)
	}

	want := "[\n{\"id\":1},\n{\"id\":2}\n]\n"
	if got := buf.String(); got != want {
		t.Fatalf("jsonWriteMessages(): want %q, have %q", want, got)
	}
}

func TestTextShortWriteMessage(t *testing.T) {
	// Exercise the compact human-readable summary paths for common message
	// shapes without depending on the local timezone.
	alice := mainContact("Alice", "+15551234567")
	bob := mainContact("Bob", "")

	tests := []struct {
		name string
		msg  signal.Message
		want string
	}{
		{
			name: "incoming body",
			msg: signal.Message{
				Type:     "incoming",
				Source:   alice,
				TimeSent: -1,
				Body:     signal.MessageBody{Text: "hello"},
			},
			want: "unknown Alice: hello\n",
		},
		{
			name: "outgoing details",
			msg: signal.Message{
				Type:        "outgoing",
				TimeSent:    -1,
				Body:        signal.MessageBody{Text: "see file"},
				Attachments: []signal.Attachment{{}},
				Edits:       []signal.Edit{{}},
				Quote: &signal.Quote{
					Recipient: bob,
					TimeSent:  -1,
				},
			},
			want: "unknown You: [reply to Bob on unknown, edited, 1 attachment] see file\n",
		},
		{
			name: "unknown message type",
			msg: signal.Message{
				Type:     "call-history",
				Source:   alice,
				TimeSent: -1,
			},
			want: "unknown Alice: [call-history message]\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			ew := errio.NewWriter(&buf)
			textShortWriteMessage(ew, &tt.msg)
			if err := ew.Err(); err != nil {
				t.Fatal(err)
			}
			if got := buf.String(); got != tt.want {
				t.Fatalf("textShortWriteMessage(): want %q, have %q", tt.want, got)
			}
		})
	}
}

func TestTextWriteMessage(t *testing.T) {
	// Cover nested quote/body rendering, where small prefix changes can make
	// exported conversations harder to read.
	alice := mainContact("Alice", "+15551234567")
	bob := mainContact("Bob", "")
	msg := signal.Message{
		Type:     "incoming",
		Source:   alice,
		TimeSent: -1,
		TimeRecv: -1,
		Body:     signal.MessageBody{Text: "hello\nthere"},
		Attachments: []signal.Attachment{
			{FileName: "photo.jpg", ContentType: "image/jpeg"},
		},
		Reactions: []signal.Reaction{
			{Recipient: bob, Emoji: "👍"},
		},
		Quote: &signal.Quote{
			Recipient: bob,
			TimeSent:  -1,
			Body:      signal.MessageBody{Text: "quoted"},
			Attachments: []signal.QuoteAttachment{
				{FileName: "clip.mp4", ContentType: "video/mp4"},
			},
		},
	}

	var buf bytes.Buffer
	ew := errio.NewWriter(&buf)
	textWriteMessage(ew, &msg)
	if err := ew.Err(); err != nil {
		t.Fatal(err)
	}

	want := "From: Alice (+15551234567)\n" +
		"Type: incoming\n" +
		"Sent: unknown\n" +
		"Received: unknown\n" +
		"Attachment: photo.jpg (image/jpeg, 0 bytes)\n" +
		"Reaction: 👍 from Bob\n" +
		"\n" +
		"> From: Bob\n" +
		"> Sent: unknown\n" +
		"> Attachment: clip.mp4 (video/mp4)\n" +
		">\n" +
		"> quoted\n" +
		"\n" +
		"hello\n" +
		"there\n" +
		"\n"
	if got := buf.String(); got != want {
		t.Fatalf("textWriteMessage(): want %q, have %q", want, got)
	}
}

func TestTextWriteMessageWithEdits(t *testing.T) {
	// Edited messages render the version history instead of the top-level body.
	msg := signal.Message{
		Type:     "outgoing",
		TimeSent: -1,
		Body:     signal.MessageBody{Text: "current"},
		Edits: []signal.Edit{
			{
				TimeEdit: -1,
				Body:     signal.MessageBody{Text: "previous"},
				Attachments: []signal.Attachment{
					{ContentType: "image/jpeg"},
				},
			},
		},
	}

	var buf bytes.Buffer
	ew := errio.NewWriter(&buf)
	textWriteMessage(ew, &msg)
	if err := ew.Err(); err != nil {
		t.Fatal(err)
	}

	got := buf.String()
	for _, want := range []string{
		"From: You\n",
		"Edited: 1 versions\n",
		"| Version: 1\n",
		"| Attachment: no filename (image/jpeg, 0 bytes)\n",
		"| Sent: unknown\n",
		"| previous\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("textWriteMessage() edit history missing %q in %q", want, got)
		}
	}
	if strings.Contains(got, "current") {
		t.Fatalf("textWriteMessage() edit history should not render current body: %q", got)
	}
}

func TestTextWriteMessages(t *testing.T) {
	// Multi-message text export starts with a conversation header.
	alice := mainContact("Alice", "+15551234567")
	msgs := []signal.Message{
		{
			Conversation: alice,
			Type:         "outgoing",
			TimeSent:     -1,
			Body:         signal.MessageBody{Text: "hello"},
		},
	}

	var buf bytes.Buffer
	ew := errio.NewWriter(&buf)
	if err := textWriteMessages(ew, msgs); err != nil {
		t.Fatal(err)
	}
	if got := buf.String(); !strings.HasPrefix(got, "Conversation: Alice (+15551234567)\n\n") {
		t.Fatalf("textWriteMessages() header: have %q", got)
	}
}

func mainContact(name, phone string) *signal.Recipient {
	// Keep main-package formatter tests from depending on signal test helpers.
	return &signal.Recipient{
		Type: signal.RecipientTypeContact,
		Contact: signal.Contact{
			Name:  name,
			Phone: phone,
		},
	}
}
