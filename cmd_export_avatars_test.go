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
	"testing"

	"github.com/tbvdm/sigtop/filename"
)

func TestAvatarFilename(t *testing.T) {
	// Avatar exports infer extensions from magic bytes instead of MIME data.
	rpt := mainContact("Alice/Bob", "+15551234567")
	opts := &avatarExportOptions{
		sanitiser: filename.NewSanitiser(filename.Unix),
	}

	tests := []struct {
		name   string
		detail string
		data   []byte
		want   string
	}{
		{"jpeg", "", []byte("\xff\xd8\xffrest"), "Alice_Bob (+15551234567).jpg"},
		{"png", "profile", []byte("\x89PNG\r\n\x1a\nrest"), "Alice_Bob (+15551234567) (profile).png"},
		{"webp", "", []byte("RIFFxxxxWEBPrest"), "Alice_Bob (+15551234567).webp"},
		{"unknown", "", []byte("unknown"), "Alice_Bob (+15551234567)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := avatarFilename(rpt, tt.detail, tt.data, opts); got != tt.want {
				t.Fatalf("avatarFilename(): want %q, have %q", tt.want, got)
			}
		})
	}
}
