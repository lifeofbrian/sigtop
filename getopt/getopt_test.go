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

package getopt

import "testing"

func TestArgConversions(t *testing.T) {
	// Option arguments expose small typed conversion helpers used by commands.
	arg := Arg{arg: "42", set: true}
	if !arg.Set() {
		t.Fatal("Set(): want true")
	}
	if got := arg.String(); got != "42" {
		t.Fatalf("String(): want 42, have %q", got)
	}
	if got, err := arg.Int(); err != nil || got != 42 {
		t.Fatalf("Int(): want 42 nil, have %d %v", got, err)
	}
	if got, err := arg.Int64(); err != nil || got != 42 {
		t.Fatalf("Int64(): want 42 nil, have %d %v", got, err)
	}

	farg := Arg{arg: "3.5", set: true}
	if got, err := farg.Float(); err != nil || got != 3.5 {
		t.Fatalf("Float(): want 3.5 nil, have %f %v", got, err)
	}
}

func TestParseArgs(t *testing.T) {
	// Cover bundled flags, attached arguments, and separated arguments.
	ParseArgs("ab:c:", []string{"-a", "-bvalue", "-c", "other", "rest"})

	if !Next() || Option() != 'a' {
		t.Fatalf("first option: want a, have %q", Option())
	}
	if got := OptionArg(); !got.Set() || got.String() != "" {
		t.Fatalf("first option arg: want set empty, have %+v", got)
	}

	if !Next() || Option() != 'b' {
		t.Fatalf("second option: want b, have %q", Option())
	}
	if got := OptionArg().String(); got != "value" {
		t.Fatalf("second option arg: want value, have %q", got)
	}

	if !Next() || Option() != 'c' {
		t.Fatalf("third option: want c, have %q", Option())
	}
	if got := OptionArg().String(); got != "other" {
		t.Fatalf("third option arg: want other, have %q", got)
	}

	if Next() {
		t.Fatal("Next(): want false")
	}
	if err := Err(); err != nil {
		t.Fatal(err)
	}
	if got := Args(); len(got) != 1 || got[0] != "rest" {
		t.Fatalf("Args(): want [rest], have %v", got)
	}
}

func TestParseArgsStopsAtNonOption(t *testing.T) {
	// Like traditional getopt, parsing stops at the first non-option.
	ParseArgs("a", []string{"file", "-a"})
	if Next() {
		t.Fatal("Next(): want false")
	}
	if got := Args(); len(got) != 2 || got[0] != "file" || got[1] != "-a" {
		t.Fatalf("Args(): have %v", got)
	}
}

func TestParseArgsErrors(t *testing.T) {
	// Error state suppresses Option and OptionArg so callers do not consume
	// stale parser state.
	tests := []struct {
		name string
		opts string
		args []string
	}{
		{"invalid option", "a", []string{"-b"}},
		{"missing argument", "a:", []string{"-a"}},
		{"invalid UTF-8", "a", []string{"-\xff"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ParseArgs(tt.opts, tt.args)
			if Next() {
				t.Fatal("Next(): want false")
			}
			if Err() == nil {
				t.Fatal("Err(): want error")
			}
			if Option() != 0 {
				t.Fatalf("Option(): want 0, have %q", Option())
			}
			if OptionArg().Set() {
				t.Fatalf("OptionArg(): want unset")
			}
		})
	}
}
