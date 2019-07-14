package main

import (
	"image"
	"log"
	"time"

	"github.com/golang/freetype"
	"github.com/muesli/streamdeck"
)

type ClockWidget struct {
	BaseWidget
}

func (w *ClockWidget) Update(dev *streamdeck.Device) {
	img := image.NewRGBA(image.Rect(0, 0, 72, 72))

	t := time.Now()
	hour := t.Format("15")
	min := t.Format("04")
	sec := t.Format("05")
	drawString(img, ttfBoldFont, hour, 22, freetype.Pt(-1, 20))
	drawString(img, ttfFont, min, 22, freetype.Pt(-1, 43))
	drawString(img, ttfThinFont, sec, 22, freetype.Pt(-1, 66))

	err := dev.SetImage(w.key, img)
	if err != nil {
		log.Fatal(err)
	}
}
