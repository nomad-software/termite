package ui

import (
	"testing"

	"termite/internal/config"
)

func TestChordConversion(t *testing.T) {
	for qtText, chord := range map[string]string{
		"Ctrl+Shift+C":    "ctrl+shift+c",
		"Ctrl+Shift+PgUp": "ctrl+shift+pageup",
		"Alt+Left":        "alt+left",
		"Ctrl+=":          "ctrl+equal",
		"Ctrl+-":          "ctrl+minus",
		"Shift+Ins":       "shift+insert",
		"Menu":            "menu",
		"Meta+Return":     "super+enter",
		"F5":              "f5",
		"Ctrl+É":          "ctrl+é", // not ASCII: cut in half, it was lost
		"":                "",
	} {
		if got := qtToChord(qtText); got != chord {
			t.Errorf("qtToChord(%q) = %q, want %q", qtText, got, chord)
		}
		if got := chordToQt(chord); got != qtText {
			t.Errorf("chordToQt(%q) = %q, want %q", chord, got, qtText)
		}
	}
	// A shifted symbol is stored by its base key, as termite matches it.
	if got := qtToChord("Ctrl+Shift++"); got != "ctrl+shift+equal" {
		t.Errorf("Ctrl+Shift++ -> %q", got)
	}
	// + is matched by the key it is on, as termite matches key presses.
	if got := qtToChord("Ctrl++"); got != "ctrl+equal" {
		t.Errorf("Ctrl++ -> %q", got)
	}
	// Only the first chord of a sequence is used.
	if got := qtToChord("Ctrl+K, Ctrl+C"); got != "ctrl+k" {
		t.Errorf("sequence -> %q", got)
	}
	// Every default binding survives a round trip.
	for action, chord := range config.DefaultKeybindings {
		if chord == "" {
			continue
		}
		if got := qtToChord(chordToQt(chord)); got != chord {
			t.Errorf("%s: %q -> %q -> %q", action, chord, chordToQt(chord), got)
		}
	}
}

// TestShortcutsListEveryAction checks that the options dialog's shortcuts
// page lists every action once: the page saves the bindings it lists, so
// one missing from it would lose its binding when the dialog is saved.
func TestShortcutsListEveryAction(t *testing.T) {
	seen := map[string]bool{}
	for _, sa := range shortcutActions {
		if seen[sa.action] {
			t.Errorf("%s is listed twice", sa.action)
		}
		seen[sa.action] = true
		if _, ok := config.DefaultKeybindings[sa.action]; !ok {
			t.Errorf("%s is listed but is not an action", sa.action)
		}
	}
	for action := range config.DefaultKeybindings {
		if !seen[action] {
			t.Errorf("%s is not listed", action)
		}
	}
}
