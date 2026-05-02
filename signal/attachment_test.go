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

package signal

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestAttachmentsFromJSON(t *testing.T) {
	// Older database versions carry attachment metadata in message JSON.
	ctx := Context{}
	msg := Message{
		TimeSent: 123,
		TimeRecv: 456,
	}
	jatts := []attachmentJSON{
		{
			ContentType: "image/jpeg",
			FileName:    "photo.jpg",
			Pending:     true,
			attachmentFile: attachmentFile{
				Version: 2,
				Path:    "photo",
				Keys:    "keys",
				Size:    789,
			},
		},
	}

	atts := ctx.attachmentsFromJSON(&msg, jatts)
	if len(atts) != 1 {
		t.Fatalf("attachmentsFromJSON() len: want 1, have %d", len(atts))
	}
	att := atts[0]
	if att.FileName != "photo.jpg" || att.ContentType != "image/jpeg" || !att.Pending {
		t.Fatalf("attachmentsFromJSON() metadata: have %+v", att)
	}
	if att.TimeSent != msg.TimeSent || att.TimeRecv != msg.TimeRecv {
		t.Fatalf("attachmentsFromJSON() times: have %+v", att)
	}
	if att.Version != 2 || att.Path != "photo" || att.Keys != "keys" || att.Size != 789 {
		t.Fatalf("attachmentsFromJSON() file: have %+v", att.attachmentFile)
	}
}

func TestReadAttachmentErrors(t *testing.T) {
	// Pending attachments and attachment records without paths cannot be read.
	ctx := Context{}

	if _, err := ctx.readAttachment(&Attachment{Pending: true}); err == nil {
		t.Fatal("readAttachment() pending: no error")
	}
	if _, err := ctx.readAttachmentFile(&attachmentFile{}); err == nil {
		t.Fatal("readAttachmentFile() empty path: no error")
	}
}

func TestReadAttachmentFileVersion1(t *testing.T) {
	// Version 1 attachment files are stored unencrypted on disk.
	dir := t.TempDir()
	attDir := filepath.Join(dir, AttachmentDir)
	if err := os.Mkdir(attDir, 0777); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(attDir, "file.txt"), []byte("hello"), 0666); err != nil {
		t.Fatal(err)
	}

	ctx := Context{dir: dir}
	got, err := ctx.readAttachmentFile(&attachmentFile{
		Version: 1,
		Path:    "file.txt",
	})
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hello" {
		t.Fatalf("readAttachmentFile(): want %q, have %q", "hello", got)
	}
}

func TestAttachmentFilePath(t *testing.T) {
	// Paths may come from another OS, so separators are normalized locally.
	ctx := Context{dir: "signal-dir"}
	got := ctx.attachmentFilePath(`sub\file.txt`)
	want := filepath.Join("signal-dir", AttachmentDir, "sub", "file.txt")
	if got != want {
		t.Fatalf("attachmentFilePath(): want %q, have %q", want, got)
	}
}

func TestFixEditedLongMessage(t *testing.T) {
	// Signal may store the full edited body in a long-text attachment; sigtop
	// restores it and removes that synthetic attachment from the export list.
	dir := t.TempDir()
	attDir := filepath.Join(dir, AttachmentDir)
	if err := os.Mkdir(attDir, 0777); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(attDir, "long.txt"), []byte("hello world"), 0666); err != nil {
		t.Fatal(err)
	}

	ctx := Context{dir: dir}
	edit := Edit{
		Body: MessageBody{Text: "hello"},
		Attachments: []Attachment{
			{ContentType: "image/jpeg"},
			{
				ContentType: LongTextType,
				attachmentFile: attachmentFile{
					Version: 1,
					Path:    "long.txt",
					Size:    int64(len("hello world")),
				},
			},
		},
	}

	if err := ctx.fixEditedLongMessage(&edit); err != nil {
		t.Fatal(err)
	}
	if edit.Body.Text != "hello world" {
		t.Fatalf("fixEditedLongMessage() body: want %q, have %q", "hello world", edit.Body.Text)
	}
	if len(edit.Attachments) != 1 || edit.Attachments[0].ContentType != "image/jpeg" {
		t.Fatalf("fixEditedLongMessage() attachments: have %+v", edit.Attachments)
	}
}

func TestFixEditedLongMessageErrors(t *testing.T) {
	// Missing edited long-message files are reported as a domain-specific
	// best-effort failure rather than a raw os.ErrNotExist.
	ctx := Context{dir: t.TempDir()}
	edit := Edit{
		Body: MessageBody{Text: "hello"},
		Attachments: []Attachment{
			{
				ContentType: LongTextType,
				attachmentFile: attachmentFile{
					Version: 1,
					Path:    "missing.txt",
				},
			},
		},
	}

	err := ctx.fixEditedLongMessage(&edit)
	if err == nil {
		t.Fatal("fixEditedLongMessage() missing long message: no error")
	}
	if errors.Is(err, os.ErrNotExist) {
		t.Fatalf("fixEditedLongMessage() should wrap missing file in domain error, have %v", err)
	}
}
