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
	"testing"
	"time"
)

func TestParseInterval(t *testing.T) {
	// Max bounds are inclusive, so coarse timestamps expand to the last
	// nanosecond in that year/month/day/minute/second.
	tests := []struct {
		in      string
		wantMin time.Time
		wantMax time.Time
	}{
		{
			in:      "2024",
			wantMin: localTime(2024, 1, 1, 0, 0, 0, 0),
			wantMax: localTime(2024, 12, 31, 23, 59, 59, int(time.Second)-1),
		},
		{
			in:      "2024-02",
			wantMin: localTime(2024, 2, 1, 0, 0, 0, 0),
			wantMax: localTime(2024, 2, 29, 23, 59, 59, int(time.Second)-1),
		},
		{
			in:      "2024-05-03T04:05",
			wantMin: localTime(2024, 5, 3, 4, 5, 0, 0),
			wantMax: localTime(2024, 5, 3, 4, 5, 59, int(time.Second)-1),
		},
		{
			in:      "2024-05-03T04:05:06",
			wantMin: localTime(2024, 5, 3, 4, 5, 6, 0),
			wantMax: localTime(2024, 5, 3, 4, 5, 6, int(time.Second)-1),
		},
		{
			in:      "2024-05,2024-06",
			wantMin: localTime(2024, 5, 1, 0, 0, 0, 0),
			wantMax: localTime(2024, 6, 30, 23, 59, 59, int(time.Second)-1),
		},
		{
			in:      "2024-05,",
			wantMin: localTime(2024, 5, 1, 0, 0, 0, 0),
		},
		{
			in:      ",2024-05",
			wantMax: localTime(2024, 5, 31, 23, 59, 59, int(time.Second)-1),
		},
	}

	for _, tt := range tests {
		got, err := parseInterval(tt.in)
		if err != nil {
			t.Fatalf("parseInterval(%q): %v", tt.in, err)
		}
		if !got.Min.Equal(tt.wantMin) {
			t.Fatalf("parseInterval(%q) min: want %s, have %s", tt.in, tt.wantMin, got.Min)
		}
		if !got.Max.Equal(tt.wantMax) {
			t.Fatalf("parseInterval(%q) max: want %s, have %s", tt.in, tt.wantMax, got.Max)
		}
	}
}

func TestParseIntervalInvalid(t *testing.T) {
	// Invalid input should fail before any database/export filtering happens.
	tests := []string{
		"2024-5",
		"2024-13",
		"2024-02-30",
		"2024-05-03 04",
		"2024-05-03T04:61",
		"today",
	}

	for _, tt := range tests {
		if _, err := parseInterval(tt); err == nil {
			t.Fatalf("parseInterval(%q): no error", tt)
		}
	}
}

func localTime(year int, month time.Month, day, hour, min, sec, nsec int) time.Time {
	// parseTime uses time.Local, so expected values should use it too.
	return time.Date(year, month, day, hour, min, sec, nsec, time.Local)
}
