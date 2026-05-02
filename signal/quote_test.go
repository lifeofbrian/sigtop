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
	"encoding/json"
	"testing"
)

func TestParseQuoteJSON(t *testing.T) {
	// Quote parsing should resolve modern ACI authors and skip long-text
	// attachments that only carry message body content.
	alice := contact("Alice")
	ctx := testContext(map[string]*Recipient{
		"aci-alice": alice,
	}, nil, nil)
	id := json.Number("12345")

	qte, err := ctx.parseQuoteJSON(&quoteJSON{
		ID:        &id,
		AuthorACI: "aci-alice",
		Text:      "hi",
		Attachments: []quoteAttachmentJSON{
			{ContentType: LongTextType, FileName: "long.txt"},
			{ContentType: "image/jpeg", FileName: "photo.jpg"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if qte.Recipient != alice {
		t.Fatalf("parseQuoteJSON() recipient: want Alice, have %+v", qte.Recipient)
	}
	if qte.TimeSent != 12345 || qte.Body.Text != "hi" {
		t.Fatalf("parseQuoteJSON() body/time: have %+v", qte)
	}
	if len(qte.Attachments) != 1 || qte.Attachments[0].FileName != "photo.jpg" {
		t.Fatalf("parseQuoteJSON() attachments: have %+v", qte.Attachments)
	}
}

func TestParseQuoteJSONByPhoneAndMissingID(t *testing.T) {
	// Older quote JSON identifies authors by phone number and may omit the
	// referenced message timestamp.
	alice := contact("Alice")
	ctx := testContext(nil, map[string]*Recipient{
		"+15551234567": alice,
	}, nil)

	qte, err := ctx.parseQuoteJSON(&quoteJSON{
		Author: "+15551234567",
		Text:   "hi",
	})
	if err != nil {
		t.Fatal(err)
	}
	if qte.Recipient != alice {
		t.Fatalf("parseQuoteJSON() recipient: want Alice, have %+v", qte.Recipient)
	}
	if qte.TimeSent != -1 {
		t.Fatalf("parseQuoteJSON() missing ID time: want -1, have %d", qte.TimeSent)
	}
}

func TestParseQuoteJSONErrors(t *testing.T) {
	// Malformed quote metadata should stop message parsing instead of creating
	// an ambiguous partial quote.
	ctx := testContext(nil, nil, nil)
	if _, err := ctx.parseQuoteJSON(&quoteJSON{}); err == nil {
		t.Fatal("parseQuoteJSON() missing author: no error")
	}

	badID := json.Number("not-a-number")
	if _, err := ctx.parseQuoteJSON(&quoteJSON{ID: &badID, Author: "+15551234567"}); err == nil {
		t.Fatal("parseQuoteJSON() invalid ID: no error")
	}
}
