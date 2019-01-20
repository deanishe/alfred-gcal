//
// Copyright (c) 2017 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2017-11-25
//

package main

import "testing"

var validFormats = []string{
	"2017-11-25", // date strings
	"20171125",
	"7", // no units
	"-7",
	"1d", // days
	"+1d",
	"-1d",
	"2w", // weeks
	"+2w",
	"-2w",
}

var invalidFormats = []string{
	"1m",
	"2q",
	"l1d",
	"*2d",
}

func TestParseDate(t *testing.T) {
	tm, ok := parseDate("0")
	if !tm.Equal(today) || !ok {
		t.Errorf("zero format failed. tm=%v", tm)
	}

	for _, s := range validFormats {
		tm, ok := parseDate(s)
		if !ok {
			t.Errorf("error parsing valid format %q", s)
		}
		if tm.IsZero() {
			t.Errorf("zero time for valid format %q", s)
		}
	}

	for _, s := range invalidFormats {
		tm, ok := parseDate(s)
		if ok {
			t.Errorf("no error parsing invalid format %q", s)
		}
		if !tm.IsZero() {
			t.Errorf("non-zero time for invalid format %q: %v", s, tm)
		}
	}
}
