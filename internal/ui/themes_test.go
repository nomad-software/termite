package ui

import (
	"testing"

	"termite/internal/config"
)

// TestThemeIDsAvoidBuiltins checks that the user's own theme, named as a
// built-in they removed, is not saved under the built-in's id: it would
// count as built-in, and Restore Built-in Themes would overwrite it.
func TestThemeIDsAvoidBuiltins(t *testing.T) {
	mine := config.BuiltinThemes["solarized-dark"]
	mine.Name = "Solarized Dark"
	edited := map[string]config.ThemeSpec{
		config.DefaultTheme: config.BuiltinThemes[config.DefaultTheme],
		"new":               mine, // solarized-dark itself removed
	}
	out, newID := themeIDs([]string{config.DefaultTheme, "new"}, edited)
	if newID[config.DefaultTheme] != config.DefaultTheme {
		t.Errorf("a built-in theme moved to %q", newID[config.DefaultTheme])
	}
	id := newID["new"]
	if _, builtin := config.BuiltinThemes[id]; builtin {
		t.Errorf("the user's theme took the built-in id %q", id)
	}
	if out[id].Name != "Solarized Dark" || len(out) != 2 {
		t.Errorf("saved %v", out)
	}
}
