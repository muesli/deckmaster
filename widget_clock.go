package main

import (
	"image"
	"log"
	"time"

	"github.com/muesli/streamdeck"
)

type ClockWidget struct {
	BaseWidget
	mode  string
}

func (w *ClockWidget) Update(dev *streamdeck.Device) {
	const margin = 4
	size := int(dev.Pixels)
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	height := size - (margin * 2)
	pt := (float64(height) / 3.0) * 72.0 / float64(dev.DPI)
	ptfull := float64(height) * 72.0 / float64(dev.DPI)

	t := time.Now()
	hour := t.Format("15")
	min := t.Format("04")
	sec := t.Format("05")

	switch w.mode {
	case "hour":
		drawString(img, image.Rect(0, 0, size, size),
			ttfBoldFont,
			hour,
			ptfull,
			image.Pt(-1, -1))
	case "min":
		drawString(img, image.Rect(0, 0, size, size),
			ttfFont,
			min,
			ptfull,
			image.Pt(-1, -1))
	case "sec":
		drawString(img, image.Rect(0, 0, size, size),
			ttfThinFont,
			sec,
			ptfull,
			image.Pt(-1, -1))
	default:
		drawString(img, image.Rectangle{image.Pt(0, margin), image.Pt(size, margin+(height/3))},
			ttfBoldFont,
			hour,
			pt,
			image.Pt(-1, -1))
		drawString(img, image.Rectangle{image.Pt(0, (height/3)+margin), image.Pt(size, margin+(height/3)*2)},
			ttfFont,
			min,
			pt,
			image.Pt(-1, -1))
		drawString(img, image.Rectangle{image.Pt(0, (height/3)*2+margin), image.Pt(size, margin+height)},
			ttfThinFont,
			sec,
			pt,
			image.Pt(-1, -1))
	}

	err := dev.SetImage(w.key, img)
	if err != nil {
		log.Fatal(err)
	}
}
