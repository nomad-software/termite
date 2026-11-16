package main

import (
	"slices"
	"testing"
	"time"
)

func TestParseScript(t *testing.T) {
	got, err := parseScript("ls{enter}{ctrl c}{ctrl ?}{wait 50}{hex 1b5b41}x{")
	if err != nil {
		t.Fatal(err)
	}
	want := []token{{text: "ls"}, {text: "\r"}, {text: "\x03"}, {text: "\x7f"}, {wait: 50 * time.Millisecond}, {text: "\x1b[A"}, {text: "x"}, {text: "{"}}
	if !slices.Equal(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
	for _, bad := range []string{"{wait x}", "{ctrl ab}", "{hex zz}", "{nope}"} {
		if _, err := parseScript(bad); err == nil {
			t.Errorf("%s: no error", bad)
		}
	}
}
