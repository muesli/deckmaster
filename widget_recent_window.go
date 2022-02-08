package main

import (
	"fmt"
	"image"
	"os"
)

// RecentWindowWidget is a widget displaying a recently activated window.
type RecentWindowWidget struct {
	*ButtonWidget

	window    uint8
	showTitle bool

	lastID uint32
}

// NewRecentWindowWidget returns a new RecentWindowWidget.
func NewRecentWindowWidget(bw *BaseWidget, opts WidgetConfig) (*RecentWindowWidget, error) {
	var window int64
	if err := ConfigValue(opts.Config["window"], &window); err != nil {
		return nil, err
	}
	var showTitle bool
	_ = ConfigValue(opts.Config["showTitle"], &showTitle)

	widget, err := NewButtonWidget(bw, opts)
	if err != nil {
		return nil, err
	}

	return &RecentWindowWidget{
		ButtonWidget: widget,
		window:       uint8(window),
		showTitle:    showTitle,
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
func (w *RecentWindowWidget) Update() error {
	img := image.NewRGBA(image.Rect(0, 0, int(w.dev.Pixels), int(w.dev.Pixels)))

	if int(w.window) < len(recentWindows) {
		if w.lastID == recentWindows[w.window].ID {
			return nil
		}
		w.lastID = recentWindows[w.window].ID

		var name string
		if w.showTitle {
			name = recentWindows[w.window].Name
			runes := []rune(name)
			if len(runes) > 10 {
				name = string(runes[:10])
			}
		}

		w.label = name
		w.SetImage(recentWindows[w.window].Icon)
		return w.ButtonWidget.Update()
	}

	return w.render(w.dev, img)
}

// TriggerAction gets called when a button is pressed.
func (w *RecentWindowWidget) TriggerAction(hold bool) {
	if xorg == nil {
		fmt.Fprintln(os.Stderr, "xorg support is disabled!")
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
