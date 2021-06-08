package main

import (
	"fmt"
	"image"

	"github.com/muesli/streamdeck"
)

// RecentWindowWidget is a widget displaying a recently activated window.
type RecentWindowWidget struct {
	BaseWidget

	window    uint8
	showTitle bool

	lastID uint32
}

// NewRecentWindowWidget returns a new RecentWindowWidget.
func NewRecentWindowWidget(bw BaseWidget, opts WidgetConfig) (*RecentWindowWidget, error) {
	var window int64
	if err := ConfigValue(opts.Config["window"], &window); err != nil {
		return nil, err
	}
	var showTitle bool
	_ = ConfigValue(opts.Config["showTitle"], &showTitle)

	return &RecentWindowWidget{
		BaseWidget: bw,
		window:     uint8(window),
		showTitle:  showTitle,
	}, nil
}

// RequiresUpdate returns true when the widget wants to be repainted.
func (w *RecentWindowWidget) RequiresUpdate() bool {
	if int(w.window) < len(recentWindows) {
		return w.lastID != recentWindows[w.window].ID
	}

	return w.BaseWidget.RequiresUpdate()
}

// Update renders the widget.
func (w *RecentWindowWidget) Update(dev *streamdeck.Device) error {
	img := image.NewRGBA(image.Rect(0, 0, int(dev.Pixels), int(dev.Pixels)))

	if int(w.window) < len(recentWindows) {
		if w.lastID == recentWindows[w.window].ID {
			return nil
		}
		w.lastID = recentWindows[w.window].ID

		var name string
		if w.showTitle {
			name = recentWindows[w.window].Name
			if len(name) > 10 {
				name = name[:10]
			}
		}

		bw := ButtonWidget{
			BaseWidget: w.BaseWidget,
			icon:       recentWindows[w.window].Icon,
			label:      name,
		}
		return bw.Update(dev)
	}

	return w.render(dev, img)
}

// TriggerAction gets called when a button is pressed.
func (w *RecentWindowWidget) TriggerAction(hold bool) {
	if xorg == nil {
		fmt.Println("xorg support is disabled!")
		return
	}

	if int(w.window) < len(recentWindows) {
		if hold {
			_ = xorg.CloseWindow(recentWindows[w.window])
			return
		}

		_ = xorg.RequestActivation(recentWindows[w.window])
	}
}
