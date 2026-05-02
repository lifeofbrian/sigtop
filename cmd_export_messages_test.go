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
	"os"
	"path/filepath"
	"testing"

	"github.com/tbvdm/sigtop/at"
	"github.com/tbvdm/sigtop/filename"
	"github.com/tbvdm/sigtop/signal"
)

func TestConversationFile(t *testing.T) {
	// Message exports choose extensions from the selected output format.
	dir := t.TempDir()
	d, err := at.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	conv := signal.Conversation{Recipient: mainContact("Alice", "+15551234567")}
	tests := []struct {
		name   string
		format formatMode
		want   string
	}{
		{"json", formatJSON, "Alice (+15551234567).json"},
		{"text", formatText, "Alice (+15551234567).txt"},
		{"text short", formatTextShort, "Alice (+15551234567).txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &messageExportOptions{
				sanitiser:   filename.NewSanitiser(filename.Unix),
				format:      tt.format,
				incremental: true,
			}
			f, err := conversationFile(d, &conv, opts)
			if err != nil {
				t.Fatal(err)
			}
			if err := f.Close(); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(filepath.Join(dir, tt.want)); err != nil {
				t.Fatalf("conversationFile(): expected %q: %v", tt.want, err)
			}
		})
	}
}
