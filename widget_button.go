package main

import (
	"image"
	"log"
	"sync"

	"github.com/golang/freetype"
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
		img := image.NewRGBA(image.Rect(0, 0, 72, 72))
		if w.label != "" {
			_ = drawImage(img, w.icon, 48, 12, 4)
			drawString(img, ttfFont, w.label, 8, freetype.Pt(-1, img.Bounds().Dx()-6))
		} else {
			_ = drawImage(img, w.icon, 64, 4, 4)
		}

		err := dev.SetImage(w.key, img)
		if err != nil {
			log.Fatal(err)
		}
	})
}
