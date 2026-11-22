package render

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestProceduralCoverage(t *testing.T) {
	for r := rune(0x2500); r <= 0x259f; r++ {
		a, ok := drawProcedural(r, 9, 18)
		if !ok {
			t.Fatalf("%U not drawn", r)
		}
		n := 0
		for _, v := range a {
			if v > 0 {
				n++
			}
		}
		if n == 0 {
			t.Errorf("%U is empty", r)
		}
	}
}

// Lines that meet at a cell edge must line up across neighbouring cells.
func TestBoxJoins(t *testing.T) {
	w, h := 9, 18
	rowsAt := func(r rune, x int) []int {
		a, _ := drawProcedural(r, w, h)
		var ys []int
		for y := 0; y < h; y++ {
			if a[y*w+x] > 128 {
				ys = append(ys, y)
			}
		}
		return ys
	}
	colsAt := func(r rune, y int) []int {
		a, _ := drawProcedural(r, w, h)
		var xs []int
		for x := 0; x < w; x++ {
			if a[y*w+x] > 128 {
				xs = append(xs, x)
			}
		}
		return xs
	}
	pairs := [][2]rune{{'─', '┼'}, {'┼', '─'}, {'═', '╬'}, {'╔', '═'}, {'━', '╋'}, {'╭', '─'}, {'─', '╮'}}
	for _, p := range pairs {
		a, b := fmt.Sprint(rowsAt(p[0], w-1)), fmt.Sprint(rowsAt(p[1], 0))
		if a != b {
			t.Errorf("%c|%c horizontal join mismatch %s vs %s", p[0], p[1], a, b)
		}
	}
	vpairs := [][2]rune{{'│', '┼'}, {'║', '╬'}, {'╔', '║'}, {'╭', '│'}, {'┃', '╋'}}
	for _, p := range vpairs {
		a, b := fmt.Sprint(colsAt(p[0], h-1)), fmt.Sprint(colsAt(p[1], 0))
		if a != b {
			t.Errorf("%c/%c vertical join mismatch %s vs %s", p[0], p[1], a, b)
		}
	}
}

func TestDumpBox(t *testing.T) {
	if os.Getenv("DUMP") == "" {
		t.Skip()
	}
	w, h := 9, 18
	chars := "┼╔╬╭╡▒▚⣿"
	var lines [18]strings.Builder
	for _, r := range chars {
		a, _ := drawProcedural(r, w, h)
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				v := a[y*w+x]
				c := " "
				if v > 170 {
					c = "#"
				} else if v > 60 {
					c = "+"
				} else if v > 0 {
					c = "."
				}
				lines[y].WriteString(c)
			}
			lines[y].WriteString("|")
		}
	}
	for _, l := range lines {
		fmt.Println(l.String())
	}
}

// TestBlockHalves checks that the half blocks split the cell where the
// quadrants do, at odd sizes too: ▌ is ▖ and ▘, ▐ is ▗ and ▝, ▀ is ▘
// and ▝, ▄ is ▖ and ▗, so a column of them lines up.
func TestBlockHalves(t *testing.T) {
	for _, size := range [][2]int{{9, 19}, {8, 18}, {7, 15}} {
		w, h := size[0], size[1]
		union := func(a, b rune) []byte {
			pa, _ := drawProcedural(a, w, h)
			pb, _ := drawProcedural(b, w, h)
			out := make([]byte, len(pa))
			for i := range pa {
				out[i] = max(pa[i], pb[i])
			}
			return out
		}
		for half, quads := range map[rune][2]rune{
			'▌': {'▖', '▘'}, '▐': {'▗', '▝'}, '▀': {'▘', '▝'}, '▄': {'▖', '▗'},
		} {
			got, _ := drawProcedural(half, w, h)
			if want := union(quads[0], quads[1]); string(got) != string(want) {
				t.Errorf("%c at %dx%d is not %c with %c", half, w, h, quads[0], quads[1])
			}
		}
	}
}
