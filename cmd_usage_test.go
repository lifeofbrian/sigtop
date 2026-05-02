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

func TestCommandUsageStatus(t *testing.T) {
	// Wrong positional argument counts should return cmdUsage before commands
	// try to open Signal data, unveil paths, or pledge capabilities.
	tests := []struct {
		name string
		exec func([]string) cmdStatus
		args []string
	}{
		{"check database", cmdCheckDatabase, []string{"extra"}},
		{"export avatars", cmdExportAvatars, []string{"one", "two"}},
		{"export attachments", cmdExportAttachments, []string{"one", "two"}},
		{"export database missing file", cmdExportDatabase, nil},
		{"export database too many files", cmdExportDatabase, []string{"one", "two"}},
		{"export key", cmdExportKey, []string{"one", "two"}},
		{"export messages", cmdExportMessages, []string{"one", "two"}},
		{"import key", cmdImportKey, []string{"one", "two"}},
		{"query database missing query", cmdQueryDatabase, nil},
		{"query database too many queries", cmdQueryDatabase, []string{"one", "two"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.exec(tt.args); got != cmdUsage {
				t.Fatalf("status: want cmdUsage, have %v", got)
			}
		})
	}
}
