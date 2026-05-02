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
	"testing"

	"github.com/tbvdm/sigtop/sqlcipher"
)

func TestQueryDatabase(t *testing.T) {
	// QueryDatabase walks multiple SQL statements and returns textified columns.
	db := memoryDB(t)
	defer db.Close()
	ctx := Context{db: db}

	rows, err := ctx.QueryDatabase("SELECT 1, 'one'; SELECT 2, 'two'")
	if err != nil {
		t.Fatal(err)
	}
	want := [][]string{{"1", "one"}, {"2", "two"}}
	if len(rows) != len(want) {
		t.Fatalf("QueryDatabase() row len: want %d, have %d", len(want), len(rows))
	}
	for i := range want {
		for j := range want[i] {
			if rows[i][j] != want[i][j] {
				t.Fatalf("QueryDatabase()[%d][%d]: want %q, have %q", i, j, want[i][j], rows[i][j])
			}
		}
	}
}

func TestDatabaseVersion(t *testing.T) {
	// Database export preserves Signal's user_version pragma.
	db := memoryDB(t)
	defer db.Close()

	if err := setDatabaseVersion(db, "main", 1234); err != nil {
		t.Fatal(err)
	}
	got, err := databaseVersion(db)
	if err != nil {
		t.Fatal(err)
	}
	if got != 1234 {
		t.Fatalf("databaseVersion(): want 1234, have %d", got)
	}
}

func TestRunPragmaCheck(t *testing.T) {
	// A clean in-memory database reports no integrity failures, while invalid
	// check names are rejected by our dispatcher.
	db := memoryDB(t)
	defer db.Close()

	results, err := runPragmaCheck(db, "integrity_check")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Fatalf("integrity_check: want no results, have %v", results)
	}

	if _, err := runPragmaCheck(db, "invalid_check"); err == nil {
		t.Fatal("runPragmaCheck() invalid: no error")
	}
}

func TestRunPragmaForeignKeyCheck(t *testing.T) {
	// foreign_key_check formats table/row details for database diagnostics.
	db := memoryDB(t)
	defer db.Close()
	if err := db.Exec(`
		PRAGMA foreign_keys = OFF;
		CREATE TABLE parent (id INTEGER PRIMARY KEY);
		CREATE TABLE child (parent_id INTEGER REFERENCES parent(id));
		INSERT INTO child VALUES (99)
	`); err != nil {
		t.Fatal(err)
	}

	results, err := runPragmaCheck(db, "foreign_key_check")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0] != "foreign key violation in row 1 of table child" {
		t.Fatalf("foreign_key_check: have %v", results)
	}
}

func TestCheckDatabase(t *testing.T) {
	// A clean in-memory database should pass the combined checks.
	db := memoryDB(t)
	defer db.Close()
	ctx := Context{db: db}

	results, err := ctx.CheckDatabase()
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Fatalf("CheckDatabase(): want no results, have %v", results)
	}
}

func memoryDB(t *testing.T) *sqlcipher.DB {
	// Use plain in-memory SQLite/SQLCipher for deterministic database helpers.
	t.Helper()
	db, err := sqlcipher.OpenFlags(":memory:", sqlcipher.OpenReadWrite|sqlcipher.OpenCreate)
	if err != nil {
		t.Fatal(err)
	}
	return db
}
