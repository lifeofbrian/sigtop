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

	"github.com/tbvdm/sigtop/filename"
	"github.com/tbvdm/sigtop/getopt"
)

func TestCommand(t *testing.T) {
	// Commands should be found by canonical name and aliases.
	tests := []struct {
		in   string
		want string
	}{
		{"export-messages", "export-messages"},
		{"msg", "export-messages"},
		{"export-attachments", "export-attachments"},
		{"att", "export-attachments"},
		{"missing", ""},
	}

	for _, tt := range tests {
		got := command(tt.in)
		if tt.want == "" {
			if got != nil {
				t.Fatalf("command(%q): want nil, have %s", tt.in, got.name)
			}
			continue
		}
		if got == nil || got.name != tt.want {
			t.Fatalf("command(%q): want %q, have %+v", tt.in, tt.want, got)
		}
	}
}

func TestEncryptionKeyFromArgument(t *testing.T) {
	// Key files may carry an explicit source system prefix.
	file := filepath.Join(t.TempDir(), "key")
	if err := os.WriteFile(file, []byte("secret\nignored\n"), 0600); err != nil {
		t.Fatal(err)
	}

	key, err := encryptionKeyFromArgument(optionArg(t, "k:", "-k", "linux:"+file))
	if err != nil {
		t.Fatal(err)
	}
	if key.OS != "linux" || string(key.Key) != "secret" {
		t.Fatalf("encryptionKeyFromArgument(): have %+v", key)
	}

	key, err = encryptionKeyFromArgument(getopt.Arg{})
	if err != nil {
		t.Fatal(err)
	}
	if key != nil {
		t.Fatalf("encryptionKeyFromArgument() unset: want nil, have %+v", key)
	}
}

func TestArgumentHelpers(t *testing.T) {
	// The command wrappers share small helpers for optional typed arguments.
	ival, err := intervalFromArgument(optionArg(t, "s:", "-s", "2024-05"))
	if err != nil {
		t.Fatal(err)
	}
	if ival.Min.IsZero() || ival.Max.IsZero() {
		t.Fatalf("intervalFromArgument(): have %+v", ival)
	}

	ival, err = intervalFromArgument(getopt.Arg{})
	if err != nil {
		t.Fatal(err)
	}
	if !ival.Min.IsZero() || !ival.Max.IsZero() {
		t.Fatalf("intervalFromArgument() unset: have %+v", ival)
	}

	s, err := filenameSanitiserFromArgument(optionArg(t, "S:", "-S", "unix"))
	if err != nil {
		t.Fatal(err)
	}
	if got := s.Sanitise("a/b"); got != "a_b" {
		t.Fatalf("filenameSanitiserFromArgument(): sanitise want a_b, have %q", got)
	}

	if _, err := filenameSanitiserFromArgument(optionArg(t, "S:", "-S", "invalid")); err == nil {
		t.Fatal("filenameSanitiserFromArgument() invalid: no error")
	}
}

func TestRecipientFilename(t *testing.T) {
	// Recipient-derived filenames should include optional details/extensions
	// before sanitising.
	s := filename.NewSanitiser(filename.Unix)
	rpt := mainContact("Alice/Bob", "+15551234567")

	if got, want := recipientFilename(rpt, ".txt", s), "Alice_Bob (+15551234567).txt"; got != want {
		t.Fatalf("recipientFilename(): want %q, have %q", want, got)
	}
	if got, want := recipientFilenameWithDetail(rpt, "profile", ".jpg", s), "Alice_Bob (+15551234567) (profile).jpg"; got != want {
		t.Fatalf("recipientFilenameWithDetail(): want %q, have %q", want, got)
	}
}

func optionArg(t *testing.T, opts string, args ...string) getopt.Arg {
	t.Helper()
	getopt.ParseArgs(opts, args)
	if !getopt.Next() {
		t.Fatalf("cannot parse option from %v: %v", args, getopt.Err())
	}
	return getopt.OptionArg()
}
