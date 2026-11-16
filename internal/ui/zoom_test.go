package ui

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/nomad-software/qute/qtcore"
	"github.com/nomad-software/qute/qutetest"
)

func TestShares(t *testing.T) {
	for _, c := range []struct {
		sizes []int
		n     int
		want  []float64
	}{
		{[]int{100, 300}, 2, []float64{0.25, 0.75}},
		{[]int{0, 0}, 2, []float64{0.5, 0.5}}, // never laid out
		{[]int{100}, 2, []float64{0.5, 0.5}},  // not one per child
		{[]int{50, 0, 50}, 3, []float64{0.5, 0, 0.5}},
	} {
		if got := shares(c.sizes, c.n); !slices.Equal(got, c.want) {
			t.Errorf("shares(%v, %d) = %v, want %v", c.sizes, c.n, got, c.want)
		}
	}
}

// zoomTab opens a window with three panes in its tab, A | (B / C), the
// focus on C.
func zoomTab(t *testing.T) (*App, *Window, *Tab) {
	t.Helper()
	a, w := newTestApp(t)
	a.runAction(focusedPane(t, w), "split_vertical")
	qutetest.Wait(150)
	a.runAction(focusedPane(t, w), "split_horizontal")
	qutetest.Wait(150)
	tab := w.list[0]
	if n := len(tab.Panes()); n != 3 {
		t.Fatalf("%d panes, want 3", n)
	}
	return a, w, tab
}

// visible lists which of the tab's panes are shown.
func visible(tab *Tab) []bool {
	var out []bool
	for _, p := range tab.Panes() {
		out = append(out, p.Widget.IsVisible())
	}
	return out
}

func TestZoomPane(t *testing.T) {
	needGUI(t)
	qutetest.Run(t, func() {
		a, w, tab := zoomTab(t)
		panes := tab.Panes()
		zp := focusedPane(t, w)
		if zp != panes[2] {
			t.Fatal("the focus is not on the last pane")
		}
		rootSizes, innerSizes := tab.root.split.Sizes(), tab.root.children[1].split.Sizes()
		hidden := panes[0]
		cols, rows := hidden.cols, hidden.rows

		qutetest.Key(zp.GL, qtcore.Key_Z, qtcore.ControlModifier|qtcore.ShiftModifier)
		// The minimap waits for the pane to fill the tab.
		if tab.zoom == nil || tab.zoom.mini.IsVisible() {
			t.Fatal("the minimap is shown before the pane is laid out again")
		}
		if !qutetest.WaitFor(func() bool { return tab.zoom.mini.IsVisible() }, 2*time.Second) {
			t.Fatal("no minimap")
		}
		if zp.Widget.Width() < tab.container.Width()-2 {
			t.Fatalf("the zoomed pane is %d wide in a tab %d wide", zp.Widget.Width(), tab.container.Width())
		}
		if tab.zoomed() != zp || !slices.Equal(visible(tab), []bool{false, false, true}) {
			t.Fatalf("zoomed %v, visible %v", tab.zoomed() == zp, visible(tab))
		}
		if got := zp.header.Text(); !strings.Contains(got, "2 panes hidden") {
			t.Errorf("header %q does not count the hidden panes", got)
		}
		if got := w.tabs.TabText(0); !strings.HasPrefix(got, zoomMark) {
			t.Errorf("tab title %q does not start with %s", got, zoomMark)
		}
		if !w.toolbar.actions["zoom_pane"].IsChecked() {
			t.Error("the toolbar's Zoom is not checked")
		}
		mini := tab.zoom.mini
		// It sits in the pane's top right corner, below the header.
		g := mini.Geometry()
		if g.Y < zp.header.Height() || g.X+g.Width < zp.Widget.Width()-zoomMapWidth || g.X < zp.Widget.Width()/2 {
			t.Errorf("minimap at %+v in a pane %d wide", g, zp.Widget.Width())
		}
		if focusedPane(t, w) != zp {
			t.Error("the zoomed pane lost the focus")
		}
		// The hidden panes are not resized.
		qutetest.Wait(100)
		if hidden.cols != cols || hidden.rows != rows {
			t.Errorf("a hidden pane went from %dx%d to %dx%d", cols, rows, hidden.cols, hidden.rows)
		}
		// A layout saved now has the sizes from before the zoom.
		if got := tab.root.save().Sizes; !slices.Equal(got, rootSizes) {
			t.Errorf("saved sizes %v, want %v", got, rootSizes)
		}

		// Again: all back, as they were.
		qutetest.Key(zp.GL, qtcore.Key_Z, qtcore.ControlModifier|qtcore.ShiftModifier)
		qutetest.Wait(100)
		if tab.zoom != nil || !slices.Equal(visible(tab), []bool{true, true, true}) {
			t.Fatalf("after unzooming: zoom %v, visible %v", tab.zoom != nil, visible(tab))
		}
		if got := tab.root.split.Sizes(); !slices.Equal(got, rootSizes) {
			t.Errorf("outer sizes %v, want %v", got, rootSizes)
		}
		if got := tab.root.children[1].split.Sizes(); !slices.Equal(got, innerSizes) {
			t.Errorf("inner sizes %v, want %v", got, innerSizes)
		}
		if mini.Alive() && mini.IsVisible() {
			t.Error("the minimap is still shown")
		}
		if strings.Contains(zp.header.Text(), "hidden") || strings.HasPrefix(w.tabs.TabText(0), zoomMark) {
			t.Error("the header or tab title still shows the zoom")
		}
		if w.toolbar.actions["zoom_pane"].IsChecked() {
			t.Error("the toolbar's Zoom is still checked")
		}

		// The toolbar's button zooms too.
		w.toolbar.actions["zoom_pane"].Trigger()
		if tab.zoomed() != zp || !w.toolbar.actions["zoom_pane"].IsChecked() {
			t.Fatal("the toolbar's Zoom did not zoom")
		}
		// A click on another pane on the map zooms that one.
		qutetest.Wait(100)
		m := tab.zoom.mini
		var cell mapCell
		for _, c := range tab.mapCells(m.inner()) {
			if c.p == panes[0] {
				cell = c
			}
		}
		qutetest.ClickAt(m, qtcore.LeftButton, qtcore.QPoint{X: int(cell.r.x + cell.r.w/2), Y: int(cell.r.y + cell.r.h/2)})
		if !qutetest.WaitFor(func() bool { return tab.zoomed() == panes[0] && focusedPane(t, w) == panes[0] }, 2*time.Second) {
			t.Fatalf("clicking the map: zoomed %v, focus on it %v", tab.zoomed() == panes[0], focusedPane(t, w) == panes[0])
		}
		if !slices.Equal(visible(tab), []bool{true, false, false}) {
			t.Errorf("visible %v", visible(tab))
		}
		// Moving the focus to the right takes the zoom there.
		a.runAction(panes[0], "focus_right")
		qutetest.Wait(100)
		right := focusedPane(t, w)
		if right == panes[0] {
			t.Fatal("focus_right did not move the focus")
		}
		if tab.zoomed() != right || visible(tab)[0] || !right.Widget.IsVisible() {
			t.Errorf("focus_right: zoom on the new pane %v, visible %v", tab.zoomed() == right, visible(tab))
		}
		if !qutetest.WaitFor(func() bool { return tab.zoom != nil && tab.zoom.mini.IsVisible() }, 2*time.Second) {
			t.Error("no minimap after focus_right")
		}
		// Unzooming from there puts every pane back as it was.
		a.runAction(right, "zoom_pane")
		qutetest.Wait(100)
		if tab.zoom != nil || slices.Contains(visible(tab), false) {
			t.Errorf("after unzooming: zoom %v, visible %v", tab.zoom != nil, visible(tab))
		}
		if got := tab.root.split.Sizes(); !slices.Equal(got, rootSizes) {
			t.Errorf("outer sizes after focus_right and unzoom %v, want %v", got, rootSizes)
		}
		if got := tab.root.children[1].split.Sizes(); !slices.Equal(got, innerSizes) {
			t.Errorf("inner sizes after focus_right and unzoom %v, want %v", got, innerSizes)
		}

		// Splitting undoes the zoom.
		a.runAction(panes[0], "zoom_pane")
		a.runAction(panes[0], "split_horizontal")
		qutetest.Wait(150)
		if tab.zoom != nil || len(tab.Panes()) != 4 || slices.Contains(visible(tab), false) {
			t.Errorf("after a split: zoom %v, %d panes, visible %v", tab.zoom != nil, len(tab.Panes()), visible(tab))
		}
	})
}

// TestZoomPaneCloses checks that a hidden pane closing keeps the zoom, and
// the zoomed pane closing shows the rest.
func TestZoomPaneCloses(t *testing.T) {
	needGUI(t)
	qutetest.Run(t, func() {
		a, w, tab := zoomTab(t)
		panes := tab.Panes()
		zp := panes[2]
		a.runAction(zp, "zoom_pane")
		qutetest.WaitFor(func() bool { return zp.Widget.Width() >= tab.container.Width()-2 }, 2*time.Second)
		qutetest.Wait(100)
		// The zoomed pane's terminal keeps its size while a hidden pane
		// closes: its program sees no resize.
		cols, rows := zp.cols, zp.rows
		var sizes []string
		onResize := zp.GL.OnResize
		zp.GL.OnResize = func(w, h int) {
			onResize(w, h)
			sizes = append(sizes, fmt.Sprintf("%dx%d", zp.cols, zp.rows))
		}
		panes[0].S.WriteString("exit\r")
		if !qutetest.WaitFor(func() bool { return len(tab.Panes()) == 2 }, 5*time.Second) {
			t.Fatal("the hidden pane did not close")
		}
		if tab.zoomed() != zp || !slices.Equal(visible(tab), []bool{false, true}) {
			t.Fatalf("zoomed %v, visible %v", tab.zoomed() == zp, visible(tab))
		}
		if got := zp.header.Text(); !strings.Contains(got, "1 pane hidden") {
			t.Errorf("header %q", got)
		}
		qutetest.Wait(100)
		want := fmt.Sprintf("%dx%d", cols, rows)
		for _, s := range sizes {
			if s != want {
				t.Errorf("the zoomed terminal went through %v, want it kept at %s", sizes, want)
				break
			}
		}
		zp.GL.OnResize = onResize
		if focusedPane(t, w) != zp {
			t.Error("the zoomed pane lost the focus")
		}

		zp.S.WriteString("exit\r")
		if !qutetest.WaitFor(func() bool { return len(tab.Panes()) == 1 }, 5*time.Second) {
			t.Fatal("the zoomed pane did not close")
		}
		if tab.zoom != nil || !tab.Panes()[0].Widget.IsVisible() {
			t.Error("the last pane is hidden")
		}
		if strings.HasPrefix(w.tabs.TabText(0), zoomMark) {
			t.Error("the tab title still shows the zoom")
		}
		// One pane: nothing to zoom.
		a.runAction(tab.Panes()[0], "zoom_pane")
		if tab.zoom != nil || w.toolbar.actions["zoom_pane"].IsChecked() {
			t.Error("a tab of one pane zoomed")
		}
	})
}

// TestZoomDuringDrag checks that zooming while a header is dragged ends the
// drag, leaving the panes where they were.
func TestZoomDuringDrag(t *testing.T) {
	needGUI(t)
	qutetest.Run(t, func() {
		a, _, tab := zoomTab(t)
		panes := tab.Panes()
		num := map[*Pane]int{}
		for i, p := range panes {
			num[p] = i
		}
		shape := layoutShape(tab.root, num)
		target := widgetRect(panes[0].Widget)
		dragHeader(panes[2], target.x+10, target.y+target.h/2, func() {
			a.runAction(panes[2], "zoom_pane")
			if tab.dragging != nil || (tab.dropZone != nil && tab.dropZone.IsVisible()) {
				t.Error("the drag goes on while zoomed")
			}
		})
		if got := layoutShape(tab.root, num); got != shape || tab.zoomed() != panes[2] {
			t.Errorf("layout %s, want %s; zoomed %v", got, shape, tab.zoomed() == panes[2])
		}
	})
}

// TestZoomMapFollowsTabShape checks that the minimap is shaped like the
// tab when only the window's height changes.
func TestZoomMapFollowsTabShape(t *testing.T) {
	needGUI(t)
	qutetest.Run(t, func() {
		a, w, tab := zoomTab(t)
		w.w.Resize(900, 700)
		qutetest.Wait(150)
		a.runAction(tab.Panes()[2], "zoom_pane")
		qutetest.Wait(100)
		m := tab.zoom.mini
		tall := m.Height()
		w.w.Resize(900, 400)
		if !qutetest.WaitFor(func() bool { return m.Height() < tall }, 2*time.Second) {
			t.Errorf("the minimap stayed %d high in a lower window", m.Height())
		}
	})
}

// TestZoomFocusAndReload checks that moving the focus out of a zoomed pane
// where there is no pane keeps the zoom, that a click on the map and
// Alt+Arrows hand the zoom and the focus straight over without passing
// through the panes being hidden, and that reloading the config keeps the
// zoom shown.
func TestZoomFocusAndReload(t *testing.T) {
	needGUI(t)
	qutetest.Run(t, func() {
		a, w, tab := zoomTab(t)
		panes := tab.Panes() // A | (B / C)
		a.runAction(panes[0], "zoom_pane")
		qutetest.WaitFor(func() bool { return tab.zoom.mini.IsVisible() }, 2*time.Second)
		a.runAction(panes[0], "focus_left") // A is leftmost
		if tab.zoomed() != panes[0] {
			t.Fatal("focus_left with nothing to the left undid the zoom")
		}

		// Clicking B on the map: focus goes from A to B and nowhere else.
		var focused []*Pane
		for _, p := range panes {
			onEvent := p.GL.OnEvent
			p.GL.OnEvent = func(ev *qtcore.QEvent) bool {
				if ev.Type() == qtcore.QEventFocusIn {
					focused = append(focused, p)
				}
				return onEvent(ev)
			}
		}
		m := tab.zoom.mini
		for _, c := range tab.mapCells(m.inner()) {
			if c.p == panes[1] {
				qutetest.ClickAt(m, qtcore.LeftButton, qtcore.QPoint{X: int(c.r.x + c.r.w/2), Y: int(c.r.y + c.r.h/2)})
			}
		}
		qutetest.WaitFor(func() bool { return tab.zoomed() == panes[1] }, 2*time.Second)
		qutetest.Wait(100)
		if !slices.Equal(focused, []*Pane{panes[1]}) {
			names := map[*Pane]string{panes[0]: "A", panes[1]: "B", panes[2]: "C"}
			var got []string
			for _, p := range focused {
				got = append(got, names[p])
			}
			t.Errorf("the focus went to %v, want only B", got)
		}
		if a.focused != panes[1] || tab.focused != panes[1] {
			t.Error("B is not the focused pane")
		}

		// focus_down from B takes the zoom to C, the focus going straight
		// there.
		focused = nil
		a.runAction(panes[1], "focus_down")
		qutetest.Wait(100)
		if tab.zoomed() != panes[2] || focusedPane(t, w) != panes[2] {
			t.Errorf("focus_down: zoom on C %v, focus on C %v", tab.zoomed() == panes[2], focusedPane(t, w) == panes[2])
		}
		if !slices.Equal(visible(tab), []bool{false, false, true}) {
			t.Errorf("focus_down: visible %v", visible(tab))
		}
		if !slices.Equal(focused, []*Pane{panes[2]}) {
			t.Errorf("focus_down passed the focus to %d panes, want only C", len(focused))
		}

		a.runAction(panes[2], "reload_config")
		qutetest.Wait(100)
		if tab.zoomed() != panes[2] || !strings.Contains(panes[2].header.Text(), "hidden") || !tab.zoom.mini.IsVisible() {
			t.Error("reloading the config lost the zoom's badge or map")
		}
		if !w.toolbar.actions["zoom_pane"].IsEnabled() {
			t.Error("Zoom is disabled with three panes")
		}
	})
}

// TestZoomFromSearchBar checks that zooming another pane while the focus
// is in a pane's search bar hands the focus straight to that pane, never
// passing it along the panes being hidden.
func TestZoomFromSearchBar(t *testing.T) {
	needGUI(t)
	qutetest.Run(t, func() {
		a, w, tab := zoomTab(t)
		panes := tab.Panes() // A | (B / C)
		a.runAction(panes[0], "zoom_pane")
		a.runAction(panes[0], "search")
		if !qutetest.WaitFor(func() bool { return panes[0].searchBar.HasFocus() }, 2*time.Second) {
			t.Fatal("the search bar did not take the focus")
		}
		var focused []*Pane
		for _, p := range panes {
			onEvent := p.GL.OnEvent
			p.GL.OnEvent = func(ev *qtcore.QEvent) bool {
				if ev.Type() == qtcore.QEventFocusIn {
					focused = append(focused, p)
				}
				return onEvent(ev)
			}
		}
		tab.zoomPane(panes[2])
		qutetest.Wait(100)
		if tab.zoomed() != panes[2] || focusedPane(t, w) != panes[2] {
			t.Fatalf("zoom on C %v, focus on C %v", tab.zoomed() == panes[2], focusedPane(t, w) == panes[2])
		}
		if !slices.Equal(focused, []*Pane{panes[2]}) {
			t.Errorf("the focus reached %d panes, want only C", len(focused))
		}
	})
}
