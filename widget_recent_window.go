package main

import (
	"image"
	"image/draw"
	"log"

	"github.com/muesli/streamdeck"
	"github.com/nfnt/resize"
)

// RecentWindowWidget is a widget displaying a recently activated window.
type RecentWindowWidget struct {
	BaseWidget
	window uint8

	lastClass string
}

func NewRecentWindowWidget(bw BaseWidget, opts WidgetConfig) (*RecentWindowWidget, error) {
	var window int64
	if err := ConfigValue(opts.Config["window"], &window); err != nil {
		return nil, err
	}

	return &RecentWindowWidget{
		BaseWidget: bw,
		window:     uint8(window),
	}, nil
}

// RequiresUpdate returns true when the widget wants to be repainted.
func (w *RecentWindowWidget) RequiresUpdate() bool {
	if int(w.window) < len(recentWindows) {
		return w.lastClass != recentWindows[w.window].Class
	}

	return w.BaseWidget.RequiresUpdate()
}

// Update renders the widget.
func (w *RecentWindowWidget) Update(dev *streamdeck.Device) error {
	img := image.NewRGBA(image.Rect(0, 0, int(dev.Pixels), int(dev.Pixels)))

	size := int(dev.Pixels)
	if int(w.window) < len(recentWindows) {
		if w.lastClass == recentWindows[w.window].Class {
			return nil
		}
		w.lastClass = recentWindows[w.window].Class

		icon := resize.Resize(uint(size-8), uint(size-8), recentWindows[w.window].Icon, resize.Bilinear)
		draw.Draw(img, image.Rect(4, 4, size-4, size-4), icon, image.Point{0, 0}, draw.Src)
	}

	return w.render(dev, img)
}

// TriggerAction gets called when a button is pressed.
func (w *RecentWindowWidget) TriggerAction() {
	if xorg == nil {
		log.Println("xorg support is disabled!")
		return
	}

	if int(w.window) < len(recentWindows) {
		_ = xorg.RequestActivation(recentWindows[w.window])
	}
}
