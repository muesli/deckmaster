package main

import (
	"image"
	"image/draw"
	"log"

	"github.com/muesli/streamdeck"
)

type RecentWindowWidget struct {
	BaseWidget
	window uint8
}

func (w *RecentWindowWidget) Update(dev *streamdeck.Device) {
	img := image.NewRGBA(image.Rect(0, 0, 72, 72))

	if int(w.window) < len(recentWindows) {
		draw.Draw(img, image.Rect(4, 4, 68, 68), recentWindows[w.window].Icon, image.Point{0, 0}, draw.Src)
	}

	err := dev.SetImage(w.key, img)
	if err != nil {
		log.Fatal(err)
	}
}

func (w *RecentWindowWidget) TriggerAction() {
	if int(w.window) < len(recentWindows) {
		x.RequestActivation(recentWindows[w.window])
	}
}
