package render

import (
	"testing"

	"termite/internal/term"
)

// TestOnlyCurrentMatch checks that only the current search match is shown,
// including one that wraps onto the next line.
func TestOnlyCurrentMatch(t *testing.T) {
	ms := []term.Match{
		{Start: term.Pos{Line: 1, Col: 3}, End: term.Pos{Line: 1, Col: 5}},
		{Start: term.Pos{Line: 1, Col: 8}, End: term.Pos{Line: 2, Col: 2}}, // wraps
		{Start: term.Pos{Line: 2, Col: 4}, End: term.Pos{Line: 2, Col: 5}},
	}
	for current := -1; current < len(ms); current++ {
		r := &Renderer{}
		r.view.Matches, r.view.Current = ms, current
		for abs := int64(0); abs <= 3; abs++ {
			for x := range 10 {
				want := current >= 0 && inMatch(ms[current], abs, x)
				if got := r.inCurrentMatch(abs, x); got != want {
					t.Errorf("current %d, line %d col %d: %v, want %v", current, abs, x, got, want)
				}
			}
		}
	}
}

// TestPlainRun checks that a ligature is only drawn over cells that need
// nothing drawn per cell: no cursor, no lines over or under them.
func TestPlainRun(t *testing.T) {
	arrow := func(attr term.Attr) []term.Cell {
		return []term.Cell{{Ch: '-', Attr: attr}, {Ch: '>', Attr: attr}, {Ch: ' '}}
	}
	none := make([]term.Extra, 3)
	r := &Renderer{}
	r.curX = 1
	if !r.plainRun(arrow(0), none, 0, 0, 2, false, -1, -1) {
		t.Error("a plain -> is not drawn as a ligature")
	}
	if r.plainRun(arrow(0), none, 0, 0, 2, true, -1, -1) {
		t.Error("the cursor on > is lost")
	}
	for _, attr := range []term.Attr{term.Attr(0).WithUnderline(term.ULSingle), term.AttrStrike, term.AttrOverline} {
		if r.plainRun(arrow(attr), none, 0, 0, 2, false, -1, -1) {
			t.Errorf("attribute %b drops its line", attr)
		}
	}
	linked := arrow(0)
	linked[1].Extra = 1
	if r.plainRun(linked, []term.Extra{{}, {Link: 1}, {}}, 0, 0, 2, false, -1, -1) {
		t.Error("a link loses its underline")
	}
	r.view.HoverActive = true
	r.view.Hover = term.Match{Start: term.Pos{Line: 0, Col: 1}, End: term.Pos{Line: 0, Col: 2}}
	if r.plainRun(arrow(0), none, 0, 0, 2, false, -1, -1) {
		t.Error("a hovered link loses its underline")
	}
}

// TestPlainRunPreeditAndSelection checks that no ligature is drawn under
// the IME preedit, or across the edge of the selection, whose cells take
// other colours than the ligature's first.
func TestPlainRunPreeditAndSelection(t *testing.T) {
	line := []term.Cell{{Ch: '-'}, {Ch: '>'}, {Ch: ' '}}
	none := make([]term.Extra, 3)
	r := &Renderer{}
	r.curX = 1
	if r.plainRun(line, none, 0, 0, 2, false, 1, 3) {
		t.Error("a ligature is drawn under the preedit")
	}
	if !r.plainRun(line, none, 0, 0, 2, false, 2, 3) {
		t.Error("a preedit after the run stops the ligature")
	}
	r.view.Sel = Selection{Active: true, Start: term.Pos{Col: 1}, End: term.Pos{Col: 3}}
	if r.plainRun(line, none, 0, 0, 2, false, -1, -1) {
		t.Error("a ligature is drawn across the selection's start")
	}
	r.view.Sel.Start.Col = 0
	if !r.plainRun(line, none, 0, 0, 2, false, -1, -1) {
		t.Error("a ligature wholly selected is not drawn")
	}
}

// TestInSelection checks block selections and a selection made backwards.
func TestInSelection(t *testing.T) {
	r := &Renderer{}
	r.view.Sel = Selection{Active: true, Start: term.Pos{Line: 2, Col: 5}, End: term.Pos{Line: 1, Col: 2}}
	for _, c := range []struct {
		abs  int64
		x    int
		want bool
	}{{1, 1, false}, {1, 2, true}, {1, 9, true}, {2, 4, true}, {2, 5, false}, {3, 0, false}} {
		if got := r.inSelection(c.abs, c.x); got != c.want {
			t.Errorf("stream: line %d col %d = %v, want %v", c.abs, c.x, got, c.want)
		}
	}
	r.view.Sel.Block = true
	for _, c := range []struct {
		abs  int64
		x    int
		want bool
	}{{1, 1, false}, {1, 2, true}, {1, 5, false}, {2, 3, true}, {2, 9, false}} {
		if got := r.inSelection(c.abs, c.x); got != c.want {
			t.Errorf("block: line %d col %d = %v, want %v", c.abs, c.x, got, c.want)
		}
	}
}

// TestLigatureBackgrounds checks that a ligature drawn in the background
// colour (inverse, or mixed with it, dim) only spans cells of one
// background, while plain text keeps it across backgrounds.
func TestLigatureBackgrounds(t *testing.T) {
	c := &GlyphCache{m: map[glyphKey]Sprite{}, textIndex: map[string]rune{}}
	key := glyphKey{ch: c.textID("->"), kind: kindText, cells: 2}
	c.m[key] = Sprite{W: 1, H: 1} // cached, so nothing is rasterized
	r := &Renderer{Cache: c}
	for _, tc := range []struct {
		attr term.Attr
		want int
	}{{0, 2}, {term.AttrInverse, 0}, {term.AttrDim, 0}} {
		line := []term.Cell{{Ch: '-', Bg: 1, Attr: tc.attr}, {Ch: '>', Bg: 4, Attr: tc.attr}}
		if n, _ := r.ligatureAt(line, 0, 0); n != tc.want {
			t.Errorf("attr %v, backgrounds differing: %d cells, want %d", tc.attr, n, tc.want)
		}
		line[1].Bg = 1
		if n, _ := r.ligatureAt(line, 0, 0); n != 2 {
			t.Errorf("attr %v, one background: %d cells, want 2", tc.attr, n)
		}
	}
}
