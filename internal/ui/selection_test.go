package ui

import (
	"testing"

	"termite/internal/term"
)

// TestBlockSelection checks that a block keeps both corners' columns
// whichever way it is dragged: up and to the right once lost both.
func TestBlockSelection(t *testing.T) {
	anchor := term.Pos{Line: 5, Col: 3}
	want := [2]term.Pos{{Line: 2, Col: 3}, {Line: 5, Col: 11}}
	wantDown := [2]term.Pos{{Line: 5, Col: 3}, {Line: 8, Col: 11}}
	for _, c := range []struct {
		at   term.Pos
		want [2]term.Pos
	}{
		{term.Pos{Line: 2, Col: 10}, want},                                             // up, right
		{term.Pos{Line: 8, Col: 10}, wantDown},                                         // down, right
		{term.Pos{Line: 2, Col: 1}, [2]term.Pos{{Line: 2, Col: 1}, {Line: 5, Col: 4}}}, // up, left
		{term.Pos{Line: 8, Col: 1}, [2]term.Pos{{Line: 5, Col: 1}, {Line: 8, Col: 4}}}, // down, left
		{term.Pos{Line: 5, Col: 3}, [2]term.Pos{{Line: 5, Col: 3}, {Line: 5, Col: 4}}}, // one cell
	} {
		start, end := blockSelection(anchor, c.at)
		if [2]term.Pos{start, end} != c.want {
			t.Errorf("dragged to %v: %v-%v, want %v-%v", c.at, start, end, c.want[0], c.want[1])
		}
	}
}
