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

import "testing"

func TestExtensionFromContentType(t *testing.T) {
	// Prefer the stable extensions sigtop chooses explicitly before falling
	// back to the platform MIME database.
	if err := addContentTypes(); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		in   string
		want string
	}{
		{"image/jpeg", ".jpg"},
		{"image/jpeg; charset=binary", ".jpg"},
		{"video/mp4", ".mp4"},
		{"video/mpeg", ".mpg"},
		{"audio/aac", ".aac"},
		{"audio/x-m4a", ".m4a"},
		{"audio/x-wav", ".wav"},
		{"application/x-sigtop-unknown", ""},
	}

	for _, tt := range tests {
		got, err := extensionFromContentType(tt.in)
		if err != nil {
			t.Fatalf("extensionFromContentType(%q): %v", tt.in, err)
		}
		if got != tt.want {
			t.Fatalf("extensionFromContentType(%q): want %q, have %q", tt.in, tt.want, got)
		}
	}
}
