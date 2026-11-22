package render

import "testing"

// TestAtlasAlloc checks the shelf packing: a 1px gutter between regions,
// a new shelf when a row is full, and refusals when nothing fits.
func TestAtlasAlloc(t *testing.T) {
	a := newAtlas(20, 12)
	steps := []struct {
		w, h, x, y int
		ok         bool
	}{
		{8, 4, 1, 1, true},
		{8, 3, 10, 1, true}, // same shelf, after the gutter
		{8, 4, 1, 6, true},  // too wide for the rest: next shelf, below the tallest
		{30, 1, 0, 0, false},
		{8, 5, 10, 6, true}, // beside it, as tall as the atlas allows
		{2, 2, 0, 0, false}, // the shelf is full and there is no room below
		{0, 3, 0, 0, true},  // empty: nothing to place
	}
	for i, s := range steps {
		x, y, ok := a.Alloc(s.w, s.h)
		if ok != s.ok || ok && s.w > 0 && (x != s.x || y != s.y) {
			t.Errorf("step %d: Alloc(%d, %d) = %d, %d, %v; want %d, %d, %v", i, s.w, s.h, x, y, ok, s.x, s.y, s.ok)
		}
	}
	a.Reset()
	if x, y, ok := a.Alloc(8, 5); !ok || x != 1 || y != 1 {
		t.Errorf("after Reset: %d, %d, %v", x, y, ok)
	}
}
