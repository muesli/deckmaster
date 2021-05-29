package main

import (
	"image"
	"log"
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

func (w *ButtonWidget) Update(dev *streamdeck.Device) {
	w.init.Do(func() {
		size := int(dev.Pixels)
		margin := size / 18
		height := size - (margin * 2)
		img := image.NewRGBA(image.Rect(0, 0, size, size))

		if w.label != "" {
			iconsize := int((float64(height) / 3.0) * 2.0)
			_ = drawImage(img, w.icon, iconsize, image.Pt(-1, margin))

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
			_ = drawImage(img, w.icon, height, image.Pt(-1, -1))
		}

		err := dev.SetImage(w.key, img)
		if err != nil {
			log.Fatal(err)
		}
	})
}
