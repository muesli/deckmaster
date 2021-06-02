package main

import (
	"image"
	"sync"

	"github.com/muesli/streamdeck"
)

type ButtonWidget struct {
	BaseWidget
	icon     string
	label    string
	fontsize float64

	init sync.Once
}

// Update renders the widget.
func (w *ButtonWidget) Update(dev *streamdeck.Device) error {
	var err error
	w.init.Do(func() {
		size := int(dev.Pixels)
		margin := size / 18
		height := size - (margin * 2)
		img := image.NewRGBA(image.Rect(0, 0, size, size))

		if w.label != "" {
			iconsize := int((float64(height) / 3.0) * 2.0)
			err = drawImage(img, w.icon, iconsize, image.Pt(-1, margin))

			bounds := img.Bounds()
			bounds.Min.Y += iconsize + margin
			bounds.Max.Y -= margin

			drawString(img,
				bounds,
				ttfFont,
				w.label,
				w.fontsize,
				image.Pt(-1, -1))
		} else {
			err = drawImage(img, w.icon, height, image.Pt(-1, -1))
		}

		if err != nil {
			return
		}
		err = w.render(dev, img)
	})

	return err
}
