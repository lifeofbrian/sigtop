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

package at

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestDirFileOperations(t *testing.T) {
	// Basic openat-style directory operations should stay relative to Dir.
	root := t.TempDir()
	d, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	if err := d.Mkdir("sub", 0777); err != nil {
		t.Fatal(err)
	}
	sub, err := d.OpenDir("sub")
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Close()

	f, err := sub.OpenFile("file.txt", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write([]byte("hello")); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	info, err := sub.Stat("file.txt", 0)
	if err != nil {
		t.Fatal(err)
	}
	if info.Name() != "file.txt" || info.Size() != 5 || info.IsDir() {
		t.Fatalf("Stat(): have name=%q size=%d isDir=%v", info.Name(), info.Size(), info.IsDir())
	}
}

func TestDirLinkAndUnlink(t *testing.T) {
	// Hard links and unlink are primitives export code uses without changing
	// process-wide working directory.
	root := t.TempDir()
	d, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	if err := os.WriteFile(filepath.Join(root, "src"), []byte("hello"), 0666); err != nil {
		t.Fatal(err)
	}
	if err := d.Link(d, "src", "hard", 0); err != nil {
		t.Fatal(err)
	}
	if err := d.Unlink("hard", 0); err != nil {
		t.Fatal(err)
	}
}

func TestDirSymlinkAndLstat(t *testing.T) {
	// Windows runners may not grant symlink creation privileges.
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires privileges on Windows")
	}

	root := t.TempDir()
	d, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	if err := os.WriteFile(filepath.Join(root, "src"), []byte("hello"), 0666); err != nil {
		t.Fatal(err)
	}
	if err := d.Symlink("src", "sym"); err != nil {
		t.Fatal(err)
	}
	info, err := d.Stat("sym", SymlinkNoFollow)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&fs.ModeSymlink == 0 {
		t.Fatalf("Stat() symlink mode: have %v", info.Mode())
	}
	if err := d.Unlink("sym", 0); err != nil {
		t.Fatal(err)
	}
}

func TestDirInvalidFlags(t *testing.T) {
	// Public wrappers reject unknown flags before reaching platform syscalls.
	d, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	tests := []struct {
		name string
		err  error
	}{
		{"link", d.Link(d, "a", "b", 12345)},
		{"unlink", d.Unlink("a", 12345)},
		{"stat", func() error { _, err := d.Stat("a", 12345); return err }()},
		{"utimes", d.Utimes("a", time.Now(), time.Now(), 12345)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var atErr *Error
			if !errors.As(tt.err, &atErr) || !errors.Is(tt.err, ErrInvalidFlag) {
				t.Fatalf("error: want ErrInvalidFlag, have %v", tt.err)
			}
		})
	}
}

func TestFutimes(t *testing.T) {
	// Futimes is used for exported attachment timestamps.
	f, err := os.CreateTemp(t.TempDir(), "file")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	mtime := time.Unix(123, 0)
	if err := Futimes(f, mtime, mtime); err != nil {
		t.Fatal(err)
	}
	info, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(mtime) {
		t.Fatalf("ModTime(): want %s, have %s", mtime, info.ModTime())
	}
}
