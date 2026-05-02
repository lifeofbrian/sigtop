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

package filename

import (
	"os"
	"testing"
)

func TestParseOS(t *testing.T) {
	// The command-line -S option accepts only these named sanitiser targets.
	tests := []struct {
		in   string
		want OS
		ok   bool
	}{
		{"native", Native, true},
		{"macos", Macos, true},
		{"unix", Unix, true},
		{"windows", Windows, true},
		{"plan9", Native, false},
	}

	for _, tt := range tests {
		got, err := ParseOS(tt.in)
		if (err == nil) != tt.ok {
			t.Fatalf("ParseOS(%q) error: want ok %v, have %v", tt.in, tt.ok, err)
		}
		if got != tt.want {
			t.Fatalf("ParseOS(%q): want %v, have %v", tt.in, tt.want, got)
		}
	}
}

func TestSanitiseUnix(t *testing.T) {
	// Unix only needs to avoid path separators, control characters, and magic
	// dot path names.
	s := NewSanitiser(Unix)

	tests := []struct {
		in   string
		want string
	}{
		{"", "_"},
		{".", "._"},
		{"..", ".._"},
		{"plain.txt", "plain.txt"},
		{"a" + string(os.PathSeparator) + "b", "a_b"},
		{"a\x00b", "a_b"},
		{"a\nb", "a_b"},
		{"CON", "CON"},
		{"trailing. ", "trailing. "},
	}

	for _, tt := range tests {
		if got := s.Sanitise(tt.in); got != tt.want {
			t.Fatalf("Sanitise(%q): want %q, have %q", tt.in, tt.want, got)
		}
	}
}

func TestSanitiseMacos(t *testing.T) {
	// APFS limits filenames to Unicode 9.0 assigned characters in addition to
	// the usual Unix-style restrictions.
	s := NewSanitiser(Macos)

	tests := []struct {
		in   string
		want string
	}{
		{"", "_"},
		{".", "._"},
		{"..", ".._"},
		{"plain.txt", "plain.txt"},
		{"a" + string(os.PathSeparator) + "b", "a_b"},
		{"a\x00b", "a_b"},
		{"a\nb", "a_b"},
		{"a\U0001f979b", "a_b"},
	}

	for _, tt := range tests {
		if got := s.Sanitise(tt.in); got != tt.want {
			t.Fatalf("Sanitise(%q): want %q, have %q", tt.in, tt.want, got)
		}
	}
}

func TestSanitiseWindows(t *testing.T) {
	// Windows has the broadest filename restrictions, including device names
	// and trailing dots/spaces.
	s := NewSanitiser(Windows)

	tests := []struct {
		in   string
		want string
	}{
		{"", "_"},
		{".", "._"},
		{"..", ".._"},
		{"plain.txt", "plain.txt"},
		{"a/b", "a_b"},
		{`a\b`, "a_b"},
		{`a:b*?c"d<e>f|g`, "a_b__c_d_e_f_g"},
		{"a\x00b", "a_b"},
		{"CON", "CON_"},
		{"con.txt", "con_.txt"},
		{"LPT9.log", "LPT9_.log"},
		{"COM0", "COM0_"},
		{"file.", "file._"},
		{"file ", "file _"},
	}

	for _, tt := range tests {
		if got := s.Sanitise(tt.in); got != tt.want {
			t.Fatalf("Sanitise(%q): want %q, have %q", tt.in, tt.want, got)
		}
	}
}
