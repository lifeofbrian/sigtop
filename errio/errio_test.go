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

package errio

import (
	"bytes"
	"errors"
	"testing"
)

type failingWriter struct {
	err error
}

func (w failingWriter) Write([]byte) (int, error) {
	return 0, w.err
}

func TestWriter(t *testing.T) {
	// The happy path should behave like the wrapped writer.
	var buf bytes.Buffer
	ew := NewWriter(&buf)

	if _, err := ew.Write([]byte("hello")); err != nil {
		t.Fatal(err)
	}
	if _, err := ew.Write([]byte(" world")); err != nil {
		t.Fatal(err)
	}
	if err := ew.Err(); err != nil {
		t.Fatal(err)
	}
	if got, want := buf.String(), "hello world"; got != want {
		t.Fatalf("buffer: want %q, have %q", want, got)
	}
}

func TestWriterKeepsFirstError(t *testing.T) {
	// Once a write fails, later writes should report the same stored error.
	want := errors.New("boom")
	ew := NewWriter(failingWriter{err: want})

	if _, err := ew.Write([]byte("hello")); !errors.Is(err, want) {
		t.Fatalf("first write error: want %v, have %v", want, err)
	}
	if _, err := ew.Write([]byte("again")); !errors.Is(err, want) {
		t.Fatalf("second write error: want %v, have %v", want, err)
	}
	if err := ew.Err(); !errors.Is(err, want) {
		t.Fatalf("Err(): want %v, have %v", want, err)
	}
}
