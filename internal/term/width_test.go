package term

import "testing"

// TestStringWidthAsWritten checks that StringWidth counts the cells the
// terminal gives a string, which joins no ZWJ or skin-tone sequence: the
// IME preedit is measured with it, and must match the text once committed.
func TestStringWidthAsWritten(t *testing.T) {
	for _, s := range []string{"abc", "日本", "é", "👩‍💻", "👍🏽", "🇬🇧"} {
		tm := feed(New(20, 2, 0), s)
		if got, want := StringWidth(s), tm.cur.X; got != want {
			t.Errorf("StringWidth(%q) = %d, the terminal took %d cells", s, got, want)
		}
	}
}
