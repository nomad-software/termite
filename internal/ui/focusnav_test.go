package ui

import "testing"

func TestNeighbour(t *testing.T) {
	// Layout (900x600, 3px handles):
	//   A | B over C | D
	// with B 0..300 and C 303..600 in the middle column.
	A := rect{0, 0, 300, 600}
	B := rect{303, 0, 300, 300}
	C := rect{303, 303, 300, 297}
	D := rect{606, 0, 294, 600}
	all := map[string]rect{"A": A, "B": B, "C": C, "D": D}
	names := []string{"A", "B", "C", "D"}
	pick := func(from string, probe float64, dx, dy int) string {
		var cands []rect
		var ids []string
		for _, n := range names {
			if n != from {
				cands = append(cands, all[n])
				ids = append(ids, n)
			}
		}
		if i := neighbour(all[from], probe, cands, dx, dy); i >= 0 {
			return ids[i]
		}
		return "-"
	}
	cases := []struct {
		from   string
		probe  float64
		dx, dy int
		want   string
	}{
		{"A", 100, 1, 0, "B"}, // cursor near the top: B
		{"A", 500, 1, 0, "C"}, // cursor near the bottom: C
		{"B", 50, 1, 0, "D"},  // never the far pane below
		{"C", 400, 1, 0, "D"},
		{"D", 100, -1, 0, "B"}, // back left follows the cursor line
		{"D", 450, -1, 0, "C"},
		{"B", 50, -1, 0, "A"},
		{"B", 400, 0, 1, "C"},
		{"C", 400, 0, -1, "B"},
		{"A", 100, -1, 0, "-"}, // nothing further left
		{"B", 400, 0, -1, "-"}, // nothing above B
		{"A", 100, 0, 1, "-"},  // C is below-right, not below A
	}
	for _, c := range cases {
		if got := pick(c.from, c.probe, c.dx, c.dy); got != c.want {
			t.Errorf("from %s probe %.0f dir (%d,%d): got %s, want %s", c.from, c.probe, c.dx, c.dy, got, c.want)
		}
	}
}

// Two panes stacked on the left and one wide pane to the right of only the
// top one: moving right from the bottom pane must not jump diagonally.
func TestNeighbourNoDiagonal(t *testing.T) {
	bottomLeft := rect{0, 303, 450, 297}
	topRight := rect{453, 0, 447, 300}
	if i := neighbour(bottomLeft, 400, []rect{topRight}, 1, 0); i != -1 {
		t.Fatal("moved diagonally to a pane that only touches at a corner")
	}
}
