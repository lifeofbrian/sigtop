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
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/tbvdm/sigtop/at"
	"github.com/tbvdm/sigtop/filename"
	"github.com/tbvdm/sigtop/signal"
)

func TestAttachmentFilename(t *testing.T) {
	// Attachment exports either preserve the provided filename or synthesize a
	// timestamped name from content type.
	defer silenceLog()()

	dir := t.TempDir()
	d, err := at.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	opts := &attachmentExportOptions{
		sanitiser: filename.NewSanitiser(filename.Unix),
	}

	tests := []struct {
		name string
		att  signal.Attachment
		want string
	}{
		{
			name: "explicit filename is sanitised",
			att: signal.Attachment{
				FileName: "a/b.jpg",
			},
			want: "a_b.jpg",
		},
		{
			name: "content type extension",
			att: signal.Attachment{
				ContentType: "image/jpeg",
				TimeSent:    fixedMillis(),
			},
			want: "attachment-" + time.UnixMilli(fixedMillis()).Format("2006-01-02-15-04-05") + ".jpg",
		},
		{
			name: "empty content type",
			att: signal.Attachment{
				TimeSent: fixedMillis(),
			},
			want: "attachment-" + time.UnixMilli(fixedMillis()).Format("2006-01-02-15-04-05"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := attachmentFilename(d, &tt.att, opts)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("attachmentFilename(): want %q, have %q", tt.want, got)
			}
		})
	}
}

func TestUniqueFilename(t *testing.T) {
	// Collision handling should keep incrementing before the file extension.
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "photo.jpg"), nil, 0666); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "photo-2.jpg"), nil, 0666); err != nil {
		t.Fatal(err)
	}

	d, err := at.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	got, err := uniqueFilename(d, "photo.jpg")
	if err != nil {
		t.Fatal(err)
	}
	if want := "photo-3.jpg"; got != want {
		t.Fatalf("uniqueFilename(): want %q, have %q", want, got)
	}
}

func TestReadWriteIncrementalFile(t *testing.T) {
	// Incremental export state is a simple newline-delimited set of attachment
	// IDs; order is intentionally not significant.
	d, err := at.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	exported, err := readIncrementalFile(d)
	if err != nil {
		t.Fatal(err)
	}
	if len(exported) != 0 {
		t.Fatalf("readIncrementalFile() initial len: want 0, have %d", len(exported))
	}

	exported["one"] = true
	exported["two"] = true
	if err := writeIncrementalFile(d, exported); err != nil {
		t.Fatal(err)
	}

	got, err := readIncrementalFile(d)
	if err != nil {
		t.Fatal(err)
	}
	for id := range exported {
		if !got[id] {
			t.Fatalf("readIncrementalFile(): missing %q", id)
		}
	}
}

func TestConversationDir(t *testing.T) {
	// Conversation attachment exports create one directory per recipient.
	dir := t.TempDir()
	d, err := at.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	opts := &attachmentExportOptions{
		sanitiser: filename.NewSanitiser(filename.Unix),
	}
	conv := signal.Conversation{Recipient: mainContact("Alice/Bob", "+15551234567")}
	cd, err := conversationDir(d, &conv, opts)
	if err != nil {
		t.Fatal(err)
	}
	defer cd.Close()

	if info, err := os.Stat(filepath.Join(dir, "Alice_Bob (+15551234567)")); err != nil || !info.IsDir() {
		t.Fatalf("conversationDir(): info=%+v err=%v", info, err)
	}
}

func TestFileExists(t *testing.T) {
	// fileExists distinguishes a missing file from other Stat errors.
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "exists"), nil, 0666); err != nil {
		t.Fatal(err)
	}
	d, err := at.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	got, err := fileExists(d, "exists")
	if err != nil || !got {
		t.Fatalf("fileExists() existing: want true nil, have %v %v", got, err)
	}
	got, err = fileExists(d, "missing")
	if err != nil || got {
		t.Fatalf("fileExists() missing: want false nil, have %v %v", got, err)
	}
}

func TestCopyAttachmentFailureRemovesFile(t *testing.T) {
	// If writing an attachment fails after creating the destination, the
	// partial export file should be removed.
	d, err := at.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	err = copyAttachment(&signal.Context{}, d, "pending", &signal.Attachment{Pending: true})
	if err == nil {
		t.Fatal("copyAttachment() pending: no error")
	}
	if ok, err := fileExists(d, "pending"); err != nil || ok {
		t.Fatalf("copyAttachment() cleanup: exists=%v err=%v", ok, err)
	}
}

func TestSetAttachmentModTime(t *testing.T) {
	// Exported attachment mtimes can be based on either sent or received time.
	if runtime.GOOS == "windows" {
		t.Skip("mtime precision differs on Windows runners")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "file"), nil, 0666); err != nil {
		t.Fatal(err)
	}
	d, err := at.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	att := signal.Attachment{
		TimeSent: fixedMillis(),
		TimeRecv: fixedMillis() + int64(time.Hour/time.Millisecond),
	}
	if err := setAttachmentModTime(d, "file", &att, mtimeSent); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(dir, "file"))
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(time.UnixMilli(att.TimeSent)) {
		t.Fatalf("mtime sent: want %s, have %s", time.UnixMilli(att.TimeSent), info.ModTime())
	}

	if err := setAttachmentModTime(d, "file", &att, mtimeRecv); err != nil {
		t.Fatal(err)
	}
	info, err = os.Stat(filepath.Join(dir, "file"))
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(time.UnixMilli(att.TimeRecv)) {
		t.Fatalf("mtime recv: want %s, have %s", time.UnixMilli(att.TimeRecv), info.ModTime())
	}

	if err := setAttachmentModTime(d, "file", &att, mtimeNone); err != nil {
		t.Fatal(err)
	}
}

func fixedMillis() int64 {
	// Use a fixed local timestamp so synthesized filenames stay deterministic.
	return time.Date(2024, 5, 3, 4, 5, 6, 0, time.Local).UnixMilli()
}

func silenceLog() func() {
	// Some filename cases intentionally trigger log warnings.
	w := log.Writer()
	log.SetOutput(io.Discard)
	return func() { log.SetOutput(w) }
}
