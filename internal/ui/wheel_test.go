package ui

import "testing"

// TestWheelLines checks that a notch of the wheel scrolls the configured
// lines, that the parts of a notch a high-resolution wheel sends add up
// to the same, and that a touchpad's pixels move the text by their own
// distance whatever the setting.
func TestWheelLines(t *testing.T) {
	for _, per := range []int{1, 3, 10} {
		var acc float64
		if got := wheelLines(&acc, 0, 120, 1, 20, per); got != per {
			t.Errorf("%d per notch: up scrolled %d", per, got)
		}
		if got := wheelLines(&acc, 0, -240, 1, 20, per); got != -2*per {
			t.Errorf("%d per notch: two down scrolled %d", per, got)
		}
		acc = 0
		total := 0
		for range 8 { // eight eighths of a notch
			total += wheelLines(&acc, 0, 15, 1, 20, per)
		}
		if total != per {
			t.Errorf("%d per notch: eighths scrolled %d", per, total)
		}
	}

	var acc float64
	total := 0
	for range 4 { // 4 × 15px at scale 2 over 20px cells: 6 lines
		total += wheelLines(&acc, 15, 120, 2, 20, 10)
	}
	if total != 6 {
		t.Errorf("pixels scrolled %d lines, want 6", total)
	}

	// No cell size yet: the angle is used, never a division by zero.
	acc = 0
	if got := wheelLines(&acc, 15, 120, 1, 0, 3); got != 3 {
		t.Errorf("no cell size: scrolled %d", got)
	}
}
