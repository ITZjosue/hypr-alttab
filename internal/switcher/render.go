package switcher

import (
	"fmt"
	"time"

	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
	"github.com/hashicorp/go-hclog"

	"hypr-dock/internal/hysc"
	"hypr-dock/internal/pkg/utils"
	"hypr-dock/pkg/ipc"
)

const GridCols = 4

// render builds the UI for all workspaces and windows
func (s *Switcher) render() {
	// Increment generation to invalidate previous async tasks
	s.renderGen++
	currentGen := s.renderGen

	// Clear existing widgets
	children := s.box.GetChildren()
	children.Foreach(func(item interface{}) {
		s.box.Remove(item.(gtk.IWidget))
	})

	// Serialize Wayland access
	sem := make(chan struct{}, 1)
	var currentRow *gtk.Box
	itemsInRow := 0

	// Window mode: one card per window instead of one card per workspace
	if !s.config.CycleWorkspaces {
		s.cardW = s.fitCardWidth(len(s.clients))
		s.renderWindowGrid(currentGen, sem)
		s.updateSelection()
		s.box.ShowAll()
		return
	}

	cards := 0
	for _, wsID := range s.workspaces {
		if len(s.workspaceMap[wsID]) > 0 {
			cards++
		}
	}
	s.cardW = s.fitCardWidth(cards)

	// Iterate over SORTED Workspaces (Recency order)
	for _, wsID := range s.workspaces {
		indices := s.workspaceMap[wsID]
		if len(indices) == 0 {
			continue
		}

		if itemsInRow%GridCols == 0 {
			currentRow = s.newRow()
		}
		itemsInRow++

		// Create workspace card
		s.createWorkspaceCard(wsID, indices, currentRow, currentGen, sem)
	}

	s.updateSelection()
	s.box.ShowAll()
}

// newRow appends a centered row so a partially filled last row stays centered
func (s *Switcher) newRow() *gtk.Box {
	row, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, s.config.Spacing)
	row.SetHAlign(gtk.ALIGN_CENTER)
	s.box.Add(row)
	return row
}

// fitCardWidth returns the card width that lets n cards fit inside the switcher
// area (minus padding and spacing), never larger than PreviewWidth.
func (s *Switcher) fitCardWidth(n int) int {
	cardW := s.config.PreviewWidth
	if n <= 0 || s.areaW <= 0 || s.areaH <= 0 {
		return cardW
	}

	cols := GridCols
	if n < cols {
		cols = n
	}
	rows := (n + cols - 1) / cols

	// Aspect ratio of the previews (they follow the monitor shape)
	ratio := 9.0 / 16.0
	if len(s.monitors) > 0 && s.monitors[0].Width > 0 && s.monitors[0].Height > 0 {
		ratio = float64(s.monitors[0].Height) / float64(s.monitors[0].Width)
	}

	// Title row above each preview: label line height + box spacing
	headerH := int(float64(s.config.FontSize)*0.8*1.6) + 5
	// Selected cards draw a 4px border and a 15px glow
	const border = 2 * 4

	availW := s.areaW - 2*s.config.Padding - (cols-1)*s.config.Spacing
	availH := s.areaH - 2*s.config.Padding - (rows-1)*s.config.Spacing - rows*(headerH+border)

	if byW := availW/cols - border; byW < cardW {
		cardW = byW
	}
	if byH := int(float64(availH/rows) / ratio); byH < cardW {
		cardW = byH
	}

	// Below this previews become unreadable; let the scroll window take over
	const minCardW = 120
	if cardW < minCardW {
		cardW = minCardW
	}
	return cardW
}

// createWorkspaceCard creates a card for a single workspace
func (s *Switcher) createWorkspaceCard(wsID int, indices []int, currentRow *gtk.Box, currentGen int, sem chan struct{}) {
	// Find relevant monitor for this workspace
	firstClient := s.clients[indices[0]] // Indices are already sorted by recency within WS
	mon, ok := s.monitorMap[firstClient.Monitor]
	if !ok && len(s.monitors) > 0 {
		mon = s.monitors[0]
	}

	// Calculate scale
	cardWidth := float64(s.cardW)
	if mon.Width <= 0 {
		mon.Width = 1920 // Fallback
	}
	scale := cardWidth / float64(mon.Width)
	cardHeight := float64(mon.Height) * scale

	// Workspace Container
	wsBox, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 5)

	// Header Box (Workspace info + Title)
	headerBox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 10)
	headerBox.SetHAlign(gtk.ALIGN_FILL)

	// 1. Workspace Pill
	wsNumBox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	utils.AddStyle(wsNumBox, "box { background-color: #333333; border-radius: 12px; padding: 2px 8px; }")

	wsNumLabel, _ := gtk.LabelNew(fmt.Sprintf("%d", wsID))
	utils.AddStyle(wsNumLabel, "label { color: #ffffff; font-weight: bold; font-size: 0.6em; }")
	wsNumBox.Add(wsNumLabel)
	headerBox.PackStart(wsNumBox, false, false, 0)

	// 2. App name and window title of the MOST RECENT (first) client
	titleLabel, _ := gtk.LabelNew(s.windowLabelFor(firstClient))
	titleLabel.SetHAlign(gtk.ALIGN_START)
	titleLabel.SetEllipsize(pango.ELLIPSIZE_END)
	titleLabel.SetMaxWidthChars(25)
	utils.AddStyle(titleLabel, "label { color: #ffffff; font-weight: bold; font-size: 0.8em; margin-left: 5px; }")
	headerBox.PackStart(titleLabel, true, true, 0)

	wsBox.Add(headerBox)

	// Fixed layout area representing the desktop
	fixed, _ := gtk.FixedNew()
	fixed.SetSizeRequest(int(cardWidth), int(cardHeight))
	utils.AddStyle(fixed, "fixed { background-color: rgba(60, 60, 60, 0.4); border-radius: 8px; }")
	wsBox.Add(fixed)
	currentRow.Add(wsBox)

	// Place Windows
	// CRITICAL: We iterate REVERSE to draw oldest first, newest last (on top)
	for i := len(indices) - 1; i >= 0; i-- {
		idx := indices[i]
		s.createWindowWidget(idx, mon, scale, fixed, currentGen, sem)
	}
}

// renderWindowGrid lays out one card per window, wrapping every GridCols items.
func (s *Switcher) renderWindowGrid(currentGen int, sem chan struct{}) {
	var currentRow *gtk.Box

	for idx := range s.clients {
		if idx%GridCols == 0 {
			currentRow = s.newRow()
		}
		s.createWindowCard(idx, currentRow, currentGen, sem)
	}
}

// createWindowCard builds a full-size preview card for a single window.
func (s *Switcher) createWindowCard(idx int, currentRow *gtk.Box, currentGen int, sem chan struct{}) {
	c := s.clients[idx]

	mon, ok := s.monitorMap[c.Monitor]
	if !ok && len(s.monitors) > 0 {
		mon = s.monitors[0]
	}
	if mon.Width <= 0 {
		mon.Width = 1920 // Fallback
	}
	if mon.Height <= 0 {
		mon.Height = 1080 // Fallback
	}

	// Card keeps the monitor aspect ratio so previews are not distorted
	cardW := s.cardW
	cardH := int(float64(cardW) * float64(mon.Height) / float64(mon.Width))

	cardBox, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 5)

	// Header: app display name, then the window title
	titleLabel, _ := gtk.LabelNew(s.windowLabelFor(c))
	titleLabel.SetHAlign(gtk.ALIGN_START)
	titleLabel.SetEllipsize(pango.ELLIPSIZE_END)
	titleLabel.SetMaxWidthChars(25)
	utils.AddStyle(titleLabel, "label { color: #ffffff; font-weight: bold; font-size: 0.8em; margin-left: 5px; }")
	cardBox.Add(titleLabel)

	// Preview body
	winBox, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	winBox.SetSizeRequest(cardW, cardH)

	ctx, _ := winBox.GetStyleContext()
	ctx.AddClass("switcher-item")

	overlay, _ := gtk.OverlayNew()
	winBox.PackStart(overlay, true, true, 0)

	centerBox, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	centerBox.SetVAlign(gtk.ALIGN_CENTER)
	centerBox.SetHAlign(gtk.ALIGN_CENTER)
	overlay.Add(centerBox)

	// Icon placeholder until the screenshot lands
	iconName := s.iconNameFor(c)
	iconSize := cardW / 3
	if iconSize > 128 {
		iconSize = 128
	}
	if iconSize < 16 {
		iconSize = 16
	}

	icon, _ := gtk.ImageNewFromIconName("image-loading", gtk.ICON_SIZE_DIALOG)
	icon.SetPixelSize(iconSize)
	icon.SetHAlign(gtk.ALIGN_CENTER)
	centerBox.Add(icon)

	s.loadIconAsync(icon, iconName, iconSize, currentGen)
	s.setupScreenshot(c, cardW, cardH, centerBox, overlay, icon, iconName, currentGen, sem)

	// Mouse Interaction (EventBox)
	eventBox, _ := gtk.EventBoxNew()
	eventBox.Add(winBox)
	if s.config.MouseSelection {
		eventBox.Connect("button-press-event", s.createWindowClickHandler())
		eventBox.Connect("enter-notify-event", s.createWindowHoverHandler(idx))
	}

	cardBox.Add(eventBox)
	currentRow.Add(cardBox)

	s.widgets[idx] = &winBox.Widget
}

// createWindowWidget creates a widget for a single window
func (s *Switcher) createWindowWidget(idx int, mon ipc.Monitor, scale float64, fixed *gtk.Fixed, currentGen int, sem chan struct{}) {
	c := s.clients[idx]

	// Relative coordinates
	relX := float64(c.At[0] - mon.X)
	relY := float64(c.At[1] - mon.Y)
	w := float64(c.Size[0])
	h := float64(c.Size[1])

	scaledX := int(relX * scale)
	scaledY := int(relY * scale)
	scaledW := int(w * scale)
	scaledH := int(h * scale)

	// Min size for visibility
	if scaledW < 24 {
		scaledW = 24
	}
	if scaledH < 24 {
		scaledH = 24
	}

	// Window Widget
	winBox, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	winBox.SetSizeRequest(scaledW, scaledH)

	// Apply base class
	ctx, _ := winBox.GetStyleContext()
	ctx.AddClass("switcher-item")

	// Container for content (Icon or Screenshot)
	overlay, _ := gtk.OverlayNew()
	winBox.PackStart(overlay, true, true, 0)

	// Background container (CenterBox)
	centerBox, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	centerBox.SetVAlign(gtk.ALIGN_CENTER)
	centerBox.SetHAlign(gtk.ALIGN_CENTER)
	overlay.Add(centerBox)

	// 1. Render Icon
	iconName := s.iconNameFor(c)
	iconSize := scaledW / 2
	if iconSize > 128 {
		iconSize = 128
	}
	if iconSize < 16 {
		iconSize = 16
	}

	// 2. Async Icon Load
	icon, _ := gtk.ImageNewFromIconName("image-loading", gtk.ICON_SIZE_DIALOG)
	icon.SetPixelSize(iconSize)
	icon.SetHAlign(gtk.ALIGN_CENTER)
	centerBox.Add(icon)

	// Load icon asynchronously
	s.loadIconAsync(icon, iconName, iconSize, currentGen)

	// 3. Screenshot Logic
	s.setupScreenshot(c, scaledW, scaledH, centerBox, overlay, icon, iconName, currentGen, sem)

	// 4. Mouse Interaction (EventBox)
	eventBox, _ := gtk.EventBoxNew()
	eventBox.Add(winBox)

	// Pointer selection is opt-in: a cursor resting over a card would otherwise
	// overwrite the keyboard selection while cycling.
	if s.config.MouseSelection {
		// Click to confirm
		eventBox.Connect("button-press-event", s.createWindowClickHandler())

		// Hover to select
		eventBox.Connect("enter-notify-event", s.createWindowHoverHandler(idx))
	}

	fixed.Put(eventBox, scaledX, scaledY)
	s.widgets[idx] = &winBox.Widget
}

// setupScreenshot handles the screenshot logic extracted from previous monolith
func (s *Switcher) setupScreenshot(c ipc.Client, w, h int, centerBox *gtk.Box, overlay *gtk.Overlay, icon *gtk.Image, iconName string, currentGen int, sem chan struct{}) {
	_, err := hysc.StreamNew(c.Address, hclog.Default())
	if err == nil {
		fingerprint := fmt.Sprintf("%s@%dx%d", getWindowFingerprint(c), w, h)
		cached, exists := s.screenshotCache[fingerprint]

		// Smart Caching Logic
		// If in active workspace (Index 0 of sorted workspaces), keep cache short (1s)
		// Otherwise, keep cache long (60s)
		cacheDuration := 60 * time.Second
		if len(s.workspaces) > 0 && c.Workspace.Id == s.workspaces[0] {
			cacheDuration = 1 * time.Second
		}

		cacheValid := exists && time.Since(cached.Timestamp) < cacheDuration

		if cacheValid {
			logTiming("[SCREENSHOT] Using cached screenshot for: %s", c.Address)
			glib.IdleAdd(func() {
				if s.renderGen != currentGen {
					return
				}
				centerBox.Remove(icon)
				img, _ := gtk.ImageNewFromPixbuf(cached.Pixbuf)
				img.SetHAlign(gtk.ALIGN_CENTER)
				centerBox.Add(img)

				// Add App Icon Badge (Fix for missing cache icon)
				badgeSize := 32
				if s.config.IconSize > 0 {
					badgeSize = s.config.IconSize
				} else {
					// Auto Mode
					badgeSize = w / 8
					if badgeSize < 32 {
						badgeSize = 32
					}
					if badgeSize > 96 {
						badgeSize = 96
					}
				}
				badge, _ := utils.CreateImage(iconName, badgeSize)
				badge.SetHAlign(gtk.ALIGN_CENTER)
				badge.SetVAlign(gtk.ALIGN_START)
				badge.SetMarginTop(8)

				overlay.AddOverlay(badge)
				overlay.ShowAll()
			})
		} else {
			s.capturePreviewAsync(c, w, h, centerBox, overlay, icon.ToWidget(), iconName, currentGen, sem, fingerprint)
		}
	}
}

// updateSelection updates the visual selection state of all widgets
func (s *Switcher) updateSelection() {
	for i, w := range s.widgets {
		if w == nil {
			continue
		}

		highlight := (i == s.selected)

		ctx, _ := w.GetStyleContext()
		if highlight {
			ctx.AddClass("switcher-item-selected")
		} else {
			ctx.RemoveClass("switcher-item-selected")
		}
	}
}
