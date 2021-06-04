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

// Update renders the widget.
func (w *ButtonWidget) Update(dev *streamdeck.Device) error {
	var err error
	size := int(dev.Pixels)
	margin := size / 18
	height := size - (margin * 2)
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	if w.label != "" {
		iconsize := int((float64(height) / 3.0) * 2.0)
		bounds := img.Bounds()

		if w.icon != "" {
			err = drawImage(img,
				findImage(filepath.Dir(deck.File), w.icon),
				iconsize,
				image.Pt(-1, margin))

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
		err = drawImage(img,
			findImage(filepath.Dir(deck.File), w.icon),
			height,
			image.Pt(-1, -1))
	}

	if err != nil {
		return err
	}
	return w.render(dev, img)
}
