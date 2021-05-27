package main

import (
	"image"
	"log"
	"sync"

	"github.com/muesli/streamdeck"
)

type ButtonWidget struct {
	BaseWidget
	icon  string
	label string

	init sync.Once
}

func (w *ButtonWidget) Update(dev *streamdeck.Device) {
	w.init.Do(func() {
		const margin = 4

		img := image.NewRGBA(image.Rect(0, 0, int(dev.Pixels), int(dev.Pixels)))
		if w.label != "" {
			size := int((float64(dev.Pixels-margin*2) / 3.0) * 2.0)
			_ = drawImage(img, w.icon, size, image.Pt(-1, margin))

			pt := (float64(dev.Pixels) / 3.0) * 42.0 / float64(dev.DPI)
			bounds := img.Bounds()
			bounds.Min.Y += size + margin/2
			drawString(img,
				bounds,
				ttfFont,
				w.label,
				pt,
				image.Pt(-1, -1))
		} else {
			_ = drawImage(img, w.icon, 64, image.Pt(-1, -1))
		}

		err := dev.SetImage(w.key, img)
		if err != nil {
			log.Fatal(err)
		}
	})
}
