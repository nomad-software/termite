package ui

import (
	"fmt"

	"github.com/nomad-software/qute"
	"github.com/nomad-software/qute/qtcore"
	"github.com/nomad-software/qute/qtgui"
	"github.com/nomad-software/qute/qtwidgets"
)

// Zooming a pane (zoom_pane) hides the other panes of its tab, so it fills
// the tab, and puts them back where they were when it is undone. The hidden
// panes keep their size, so their terminals are neither reflowed nor told
// of a resize. That there is more in the tab is shown three ways: the
// zoomed pane's header counts the hidden panes, the tab's title starts with
// zoomMark, and a minimap (zoomMap) in the pane's top right corner draws
// the tab's layout, the zoomed pane filled in. Clicking a pane on the map
// zooms that one instead; clicking the zoomed one undoes the zoom.
//
// Anything that changes the split tree undoes the zoom first (splitting,
// moving a pane, a pane closing), and so does moving the focus to a hidden
// pane (Alt+Arrows, focusZoomed); when a hidden pane closes the zoom is
// made again afterwards.

// zoomMark starts the title of a tab whose pane is zoomed, and the header
// of that pane.
const zoomMark = "⤢"

// zoomState is a tab's zoom.
type zoomState struct {
	pane    *Pane
	hidden  []*Node  // the pane's ancestors' other children, hidden
	parents []*Node  // the pane's ancestors, innermost first
	mini    *zoomMap // the minimap, on the pane
}

// zoomed returns the tab's zoomed pane, or nil.
func (t *Tab) zoomed() *Pane {
	if t.zoom == nil {
		return nil
	}
	return t.zoom.pane
}

// toggleZoom zooms p, or undoes the tab's zoom.
func (t *Tab) toggleZoom(p *Pane) {
	if t.zoom != nil {
		t.unzoom()
		return
	}
	t.zoomPane(p)
}

// zoomPane hides every pane of the tab but p. A tab of one pane has
// nothing to hide.
func (t *Tab) zoomPane(p *Pane) {
	t.unzoom()
	node := t.root.find(p)
	if node == nil || node.parent == nil || p.exited {
		return
	}
	t.cancelDrag() // the panes it could be dropped on are going
	// The focus goes to p before the pane that has it is hidden: Qt would
	// pass it along the tab's panes as they go, each telling its program
	// it had the focus and lost it.
	if fw := qtwidgets.QApplicationFocusWidget(); fw != nil && fw.Ptr() != p.GL.Ptr() {
		for _, q := range t.Panes() {
			if fw.Ptr() == q.GL.Ptr() {
				p.GL.SetFocus()
				break
			}
		}
	}
	z := &zoomState{pane: p}
	// Hiding a splitter's child changes that splitter's sizes only, so
	// each is noted before its own children are hidden.
	for n := node; n.parent != nil; n = n.parent {
		par := n.parent
		par.zoomSizes = par.split.Sizes()
		z.parents = append(z.parents, par)
		for _, c := range par.children {
			if c != n {
				c.widget().Hide()
				z.hidden = append(z.hidden, c)
			}
		}
	}
	t.zoom = z
	t.focused = p
	z.mini = newZoomMap(t, p)
	t.zoomChanged(p)
	// The map is shown once the pane fills the tab: Qt lays the splitters
	// out again later, and shown now it would flash over the pane at its
	// old size. Callbacks posted now run after that layout.
	qute.Post(func() {
		if t.zoom == z && !p.exited && z.mini.Alive() {
			z.mini.place()
			z.mini.Show()
			z.mini.Raise()
		}
	})
}

// unzoom shows the tab's hidden panes again, with the sizes they had.
func (t *Tab) unzoom() {
	z := t.zoom
	if z == nil {
		return
	}
	t.zoom = nil
	for _, n := range z.hidden {
		n.widget().Show()
	}
	// Outermost first, so each splitter has its own size back before its
	// children are shared out. A window resized meanwhile scales them.
	for i := len(z.parents) - 1; i >= 0; i-- {
		n := z.parents[i]
		if len(n.zoomSizes) == n.split.Count() {
			n.split.SetSizes(n.zoomSizes)
		}
		n.zoomSizes = nil
	}
	if z.mini.Alive() {
		z.mini.Hide()
		z.mini.DeleteLater()
	}
	t.zoomChanged(z.pane)
}

// focusZoomed moves the focus from p, zoomed, to its neighbour in
// direction (dx, dy), found where the panes were before the zoom, which is
// undone then. With no neighbour that way the zoom stays.
func (t *Tab) focusZoomed(p *Pane, dx, dy int) {
	var cur rect
	var others []*Pane
	var rects []rect
	for _, c := range t.mapCells(widgetRect(t.container)) {
		if c.p == p {
			cur = c.r
		} else {
			others = append(others, c.p)
			rects = append(rects, c.r)
		}
	}
	// The cursor's place across the pane, scaled to where the pane was.
	px, py := cursorPoint(p)
	r := paneRect(p)
	probe := cur.y + (py-r.y)/r.h*cur.h
	if dy != 0 {
		probe = cur.x + (px-r.x)/r.w*cur.w
	}
	i := neighbour(cur, probe, rects, dx, dy)
	if i < 0 {
		return
	}
	t.unzoom()
	others[i].GL.SetFocus()
}

// updateZoomMap redraws the minimap, if the tab has one.
func (t *Tab) updateZoomMap() {
	if t.zoom != nil && t.zoom.mini.Alive() {
		t.zoom.mini.Update()
	}
}

// zoomChanged shows the tab's zoom, or that it has gone, on p (the pane
// zoomed, or zoomed until now), the tab's title and the toolbar.
func (t *Tab) zoomChanged(p *Pane) {
	if !p.exited {
		setBoolProperty(p.header, "zoomed", t.zoomed() == p)
		p.repolishHeader() // before layoutHeader: the font may change
		p.layoutHeader()
	}
	t.win.updateTitles() // and the toolbar
}

// zoomBadge is what the header of the zoomed pane shows before its
// directory, or "" for any other pane.
func (p *Pane) zoomBadge() string {
	t := p.tab
	if t.zoomed() != p {
		return ""
	}
	n := len(t.Panes()) - 1
	s := "s"
	if n == 1 {
		s = ""
	}
	return fmt.Sprintf("%s Zoomed: %d pane%s hidden  ·  ", zoomMark, n, s)
}

// ---------------------------------------------------------------- minimap

const (
	zoomMapWidth  = 240 // logical pixels
	zoomMapMargin = 8   // from the corner of the terminal
)

// mapCell is a pane's place on the minimap.
type mapCell struct {
	p *Pane
	r rect
}

// mapCells shares r out among the tab's panes as its splitters share the
// tab, as they were before the zoom.
func (t *Tab) mapCells(r rect) []mapCell { return cellsOf(t.root, r, nil) }

func cellsOf(n *Node, r rect, out []mapCell) []mapCell {
	if n.pane != nil {
		return append(out, mapCell{n.pane, r})
	}
	fr := shares(n.sizes(), len(n.children))
	across := n.split.Orientation() == qtcore.Horizontal
	pos := 0.0
	for i, c := range n.children {
		sub := r
		if across {
			sub.x, sub.w = r.x+pos*r.w, fr[i]*r.w
		} else {
			sub.y, sub.h = r.y+pos*r.h, fr[i]*r.h
		}
		pos += fr[i]
		out = cellsOf(c, sub, out)
	}
	return out
}

// shares turns n splitter sizes into fractions of their sum: equal ones
// when the sizes are unknown (never laid out, or not n of them).
func shares(sizes []int, n int) []float64 {
	out := make([]float64, n)
	total := 0
	for _, s := range sizes {
		total += max(s, 0)
	}
	for i := range out {
		if len(sizes) != n || total == 0 {
			out[i] = 1 / float64(n)
		} else {
			out[i] = float64(max(sizes[i], 0)) / float64(total)
		}
	}
	return out
}

// zoomMap is the minimap over a zoomed pane.
type zoomMap struct {
	qtwidgets.QWidgetBase
	t *Tab
	p *Pane
}

func newZoomMap(t *Tab, p *Pane) *zoomMap {
	m := &zoomMap{t: t, p: p}
	w := qtwidgets.NewQWidgetWith(m, p.Widget)
	w.SetObjectName("termiteZoomMap")
	w.SetFocusPolicy(qtcore.NoFocus)
	setCursorShape(w, qtcore.PointingHandCursor)
	w.SetToolTip("Other panes are hidden. Click one to show it instead, or this one to show them all.")
	w.Hide() // until zoomPane shows it
	return m
}

// place puts the map in the top right corner of the pane's terminal, shaped
// like the tab. It follows the pane's size (layoutHeader, the terminal's
// resize).
func (m *zoomMap) place() {
	p := m.p
	cw, ch := m.t.container.Width(), m.t.container.Height()
	h := zoomMapWidth * 2 / 3
	if cw > 0 && ch > 0 {
		h = min(max(zoomMapWidth*ch/cw, zoomMapWidth/3), zoomMapWidth)
	}
	x := p.Widget.Width() - zoomMapWidth - zoomMapMargin
	if p.scrollbar.IsVisible() {
		x -= p.scrollbar.Width()
	}
	y := p.header.Height() + zoomMapMargin
	m.SetGeometryRect(qtcore.QRect{X: max(x, 0), Y: y, Width: zoomMapWidth, Height: h})
}

// inner is where the map draws the panes, inside its border.
func (m *zoomMap) inner() rect {
	return rect{3, 3, float64(m.Width() - 6), float64(m.Height() - 6)}
}

// focusTintAlpha is the opacity, out of 255, of the focused header's colour
// where it tints what it lies over: a drop zone, a pane on the minimap.
const focusTintAlpha = 90

// themeColor is a theme's 0xRRGGBB colour with the opacity alpha.
func themeColor(c uint32, alpha uint8) qtgui.QColor {
	return qtgui.QColor{R: uint8(c >> 16), G: uint8(c >> 8), B: uint8(c), A: alpha}
}

// PaintEvent overrides QWidget::paintEvent: the tab's layout in the pane
// header colours, the zoomed pane filled in, and a hidden pane that typing
// reaches (broadcast) tinted.
func (m *zoomMap) PaintEvent(*qtgui.QPaintEvent) {
	th := m.t.win.app.theme
	pr := qtgui.NewQPainterOn(m)
	defer pr.End()
	w, h := m.Width(), m.Height()
	pr.FillRect(qtcore.QRect{Width: w, Height: h}, themeColor(th.HeaderBg, 235))
	pr.SetBrushStyle(qtcore.NoBrush)
	pr.SetPenColor(themeColor(th.HeaderFg, 255))
	pr.DrawRect(qtcore.QRect{Width: w - 1, Height: h - 1})
	app := m.t.win.app
	for _, c := range m.t.mapCells(m.inner()) {
		r := qtcore.QRect{X: int(c.r.x) + 1, Y: int(c.r.y) + 1, Width: int(c.r.w) - 2, Height: int(c.r.h) - 2}
		switch {
		case c.p == m.t.zoomed():
			pr.FillRect(r, themeColor(th.HeaderFocusBg, 255))
		case app.broadcastsTo(c.p):
			pr.FillRect(r, themeColor(th.HeaderFocusBg, focusTintAlpha))
		}
		pr.SetPenColor(themeColor(th.HeaderFg, 200))
		pr.DrawRect(qtcore.QRect{X: r.X, Y: r.Y, Width: r.Width - 1, Height: r.Height - 1})
	}
}

// MousePressEvent overrides QWidget::mousePressEvent: a click on a pane
// zooms it, or on the zoomed pane undoes the zoom.
func (m *zoomMap) MousePressEvent(ev *qtgui.QMouseEvent) {
	if ev.Button() != qtcore.LeftButton {
		return
	}
	pos := ev.Position()
	t := m.t
	for _, c := range t.mapCells(m.inner()) {
		if pos.X < c.r.x || pos.Y < c.r.y || pos.X >= c.r.x+c.r.w || pos.Y >= c.r.y+c.r.h {
			continue
		}
		if c.p == t.zoomed() {
			t.unzoom()
		} else {
			t.zoomPane(c.p)
		}
		c.p.GL.SetFocus()
		return
	}
}

// placeZoomMap moves the minimap with p, when p is zoomed.
func (p *Pane) placeZoomMap() {
	if z := p.tab.zoom; z != nil && z.pane == p && z.mini.Alive() {
		z.mini.place()
	}
}
