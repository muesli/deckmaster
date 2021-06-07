package main

import (
	"image"
	"path/filepath"

	"github.com/muesli/streamdeck"
)

// ButtonWidget is a simple widget displaying an icon and/or label.
type ButtonWidget struct {
	BaseWidget
	icon     string
	label    string
	fontsize float64
}

func NewButtonWidget(bw BaseWidget, opts WidgetConfig) (*ButtonWidget, error) {
	bw.setInterval(opts.Interval, 0)

	var icon, label string
	ConfigValue(opts.Config["icon"], &icon)
	ConfigValue(opts.Config["label"], &label)
	var fontsize float64
	ConfigValue(opts.Config["fontsize"], &fontsize)

	return &ButtonWidget{
		BaseWidget: bw,
		icon:       icon,
		label:      label,
		fontsize:   fontsize,
	}, nil
}

// Update renders the widget.
func (w *ButtonWidget) Update(dev *streamdeck.Device) error {
	size := int(dev.Pixels)
	margin := size / 18
	height := size - (margin * 2)
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	if w.label != "" {
		iconsize := int((float64(height) / 3.0) * 2.0)
		bounds := img.Bounds()

		if w.icon != "" {
			err := drawImage(img,
				findImage(filepath.Dir(deck.File), w.icon),
				iconsize,
				image.Pt(-1, margin))

			if err != nil {
				return err
			}

			bounds.Min.Y += iconsize + margin
			bounds.Max.Y -= margin
		}

		drawString(img,
			bounds,
			ttfFont,
			w.label,
			dev.DPI,
			w.fontsize,
			image.Pt(-1, -1))
	} else if w.icon != "" {
		err := drawImage(img,
			findImage(filepath.Dir(deck.File), w.icon),
			height,
			image.Pt(-1, -1))

		if err != nil {
			return err
		}
	}

	return w.render(dev, img)
}
