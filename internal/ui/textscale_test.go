package ui

import (
	"testing"
	"time"

	"github.com/nomad-software/qute/qutetest"

	"termite/internal/config"
)

// TestPaneTextScale checks that Larger and Smaller text change the pane
// typed in alone (or every pane typing reaches), that new panes start at
// the usual size, that layouts keep each pane's size, that the config's
// font size stays the base under them and is never written, and that a
// size no pane uses any more frees its glyph cache.
func TestPaneTextScale(t *testing.T) {
	needGUI(t)
	qutetest.Run(t, func() {
		a, w := newTestApp(t)
		a.runAction(focusedPane(t, w), "split_vertical")
		qutetest.Wait(200)
		tab := w.list[0]
		pa, pb := tab.Panes()[0], tab.Panes()[1]
		if pa.R == nil || pb.R == nil {
			t.Fatal("the panes have not been drawn")
		}
		colsA, colsB := pa.cols, pb.cols

		a.runAction(pb, "font_bigger")
		if pb.fontStep != 1 || pa.fontStep != 0 {
			t.Fatalf("steps %d and %d after Larger text in the second pane", pa.fontStep, pb.fontStep)
		}
		if pb.cols >= colsB || pa.cols != colsA {
			t.Errorf("columns %d→%d (larger) and %d→%d (untouched)", colsB, pb.cols, colsA, pa.cols)
		}
		if pb.cache == pa.cache || len(a.caches) != 2 {
			t.Errorf("%d glyph caches, the larger pane's shared: %v", len(a.caches), pb.cache == pa.cache)
		}
		// Steps down and back up land on the usual size exactly, sharing
		// its cache again.
		for range 7 {
			a.runAction(pb, "font_smaller")
		}
		for range 7 {
			a.runAction(pb, "font_bigger")
		}
		if pb.fontStep != 1 || len(a.caches) != 2 {
			t.Errorf("down 7 and up 7: step %d, %d caches", pb.fontStep, len(a.caches))
		}
		for range 40 {
			a.runAction(pb, "font_smaller")
		}
		if pb.fontStep != minFontStep {
			t.Errorf("step %d after many Smaller, want %d", pb.fontStep, minFontStep)
		}
		a.runAction(pb, "font_reset")
		if pb.fontStep != 0 || pb.cols != colsB || pb.cache != pa.cache || len(a.caches) != 1 {
			t.Errorf("after Reset: step %d, %d columns, %d caches", pb.fontStep, pb.cols, len(a.caches))
		}

		// A split starts at the usual size.
		a.runAction(pb, "font_bigger")
		a.runAction(pb, "split_horizontal")
		qutetest.Wait(200)
		pc := tab.Panes()[2]
		if pc.fontStep != 0 {
			t.Errorf("a pane split from a larger one is at step %d", pc.fontStep)
		}
		// So does a new tab.
		a.runAction(pb, "new_tab")
		qutetest.Wait(200)
		if p := w.currentPane(); p.fontStep != 0 {
			t.Errorf("a new tab's pane is at step %d", p.fontStep)
		}
		w.tabs.SetCurrentIndex(0)

		// Broadcast: every pane typing reaches changes.
		a.setBroadcast(BroadcastAll)
		before := []int{pa.fontStep, pb.fontStep, pc.fontStep}
		a.runAction(pa, "font_smaller")
		a.setBroadcast(BroadcastOff)
		for i, p := range []*Pane{pa, pb, pc} {
			if p.fontStep != before[i]-1 {
				t.Errorf("pane %d: step %d, was %d, with input broadcast", i, p.fontStep, before[i])
			}
		}

		// The config's font size is the base, the panes' scales stay on
		// top, and the file never holds a scale.
		step := pb.fontStep
		if err := config.SetValues(a.cfgPath, map[string]any{"font_size": 20}); err != nil {
			t.Fatal(err)
		}
		a.runAction(pa, "reload_config")
		if pb.fontStep != step {
			t.Fatalf("after a new font size: step %d, was %d", pb.fontStep, step)
		}
		if _, ok := a.caches[cacheKey{a.cfg.FontFamily, 20, a.cfg.LineHeight, step, pb.dpr}]; !ok {
			t.Error("no glyph cache at the new size times the pane's scale")
		}
		if c, _ := config.Load(a.cfgPath); c.FontSize != 20 {
			t.Errorf("the config's font size is %v, want 20", c.FontSize)
		}

		// A layout keeps each pane's scale.
		saved := tab.root.save()
		w.loadLayout(&LayoutFile{Tabs: []LayoutNode{saved}})
		if !qutetest.WaitFor(func() bool { return len(w.list) == 1 && len(w.list[0].Panes()) == 3 }, 5*time.Second) {
			t.Fatalf("the layout did not open: %d tabs", len(w.list))
		}
		qutetest.Wait(200)
		for i, p := range w.list[0].Panes() {
			want := []int{pa.fontStep, pb.fontStep, pc.fontStep}[i]
			if p.fontStep != want {
				t.Errorf("layout pane %d: step %d, want %d", i, p.fontStep, want)
			}
		}

		// Unused sizes are freed: one cache per scale in use.
		used := map[int]bool{}
		for _, p := range w.allPanes() {
			used[p.fontStep] = true
		}
		if len(a.caches) != len(used) {
			t.Errorf("%d glyph caches for %d sizes in use", len(a.caches), len(used))
		}
	})
}
