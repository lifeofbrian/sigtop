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

package safestorage

import (
	"encoding/base64"
	"testing"
)

func TestNewAppAndSetBackend(t *testing.T) {
	// Backend names mirror the values stored in Signal/Chromium config.
	app := NewApp("Signal", "/tmp/signal")
	if app.name != "Signal" || app.dir != "/tmp/signal" {
		t.Fatalf("NewApp(): have %+v", app)
	}

	tests := []struct {
		in      string
		backend backend
	}{
		{"gnome_libsecret", backendGnome},
		{"kwallet", backendKwallet4},
		{"kwallet5", backendKwallet5},
		{"kwallet6", backendKwallet6},
	}

	for _, tt := range tests {
		if err := app.SetBackend(tt.in); err != nil {
			t.Fatal(err)
		}
		if app.backend != tt.backend {
			t.Fatalf("SetBackend(%q): want %v, have %v", tt.in, tt.backend, app.backend)
		}
	}

	if err := app.SetBackend("unknown"); err == nil {
		t.Fatal("SetBackend() unknown: no error")
	}
}

func TestSetEncryptionKey(t *testing.T) {
	// Each supported OS uses a different raw key format or derivation.
	tests := []RawEncryptionKey{
		{OS: "linux", Key: []byte("secret")},
		{OS: "macos", Key: []byte("secret")},
		{OS: "windows", Key: []byte(base64.StdEncoding.EncodeToString(make([]byte, windowsKeySize)))},
	}

	for _, tt := range tests {
		t.Run(tt.OS, func(t *testing.T) {
			app := NewApp("Signal", "/tmp/signal")
			if err := app.SetEncryptionKey(tt); err != nil {
				t.Fatal(err)
			}
			if !app.keySet {
				t.Fatal("keySet: want true")
			}
			if len(app.key) != keySize && tt.OS != "windows" {
				t.Fatalf("derived key len: want %d, have %d", keySize, len(app.key))
			}
			raw, err := app.EncryptionKey()
			if err != nil {
				t.Fatal(err)
			}
			if raw.OS != tt.OS || string(raw.Key) != string(tt.Key) {
				t.Fatalf("EncryptionKey(): want %+v, have %+v", tt, raw)
			}
		})
	}
}

func TestSetEncryptionKeyErrors(t *testing.T) {
	// Bad platform names and malformed Windows keys should fail early.
	tests := []RawEncryptionKey{
		{OS: "plan9", Key: []byte("secret")},
		{OS: "windows", Key: []byte("not-base64")},
		{OS: "windows", Key: []byte(base64.StdEncoding.EncodeToString([]byte("short")))},
	}

	for _, tt := range tests {
		app := NewApp("Signal", "/tmp/signal")
		if err := app.SetEncryptionKey(tt); err == nil {
			t.Fatalf("SetEncryptionKey(%+v): no error", tt)
		}
	}
}

func TestUnpad(t *testing.T) {
	// Linux/macOS safe-storage decryption uses PKCS-style block padding.
	got, err := unpad([]byte("abc\x01"), 16)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "abc" {
		t.Fatalf("unpad(): want abc, have %q", got)
	}

	if got, err := unpad(nil, 16); err != nil || got != nil {
		t.Fatalf("unpad(nil): want nil nil, have %v %v", got, err)
	}

	tests := [][]byte{
		[]byte("abc\x00"),
		[]byte("abc\x05"),
		[]byte("abc\x02\x03"),
	}
	for _, tt := range tests {
		if _, err := unpad(tt, 16); err == nil {
			t.Fatalf("unpad(%q): no error", tt)
		}
	}
}
