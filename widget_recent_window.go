package main

import (
	"image"
	"image/draw"
	"log"

	"github.com/muesli/streamdeck"
	"github.com/nfnt/resize"
)

type RecentWindowWidget struct {
	BaseWidget
	window uint8

	lastClass string
}

func (w *RecentWindowWidget) UpdateImage(dev *streamdeck.Device) error {
	img := image.NewRGBA(image.Rect(0, 0, int(dev.Pixels), int(dev.Pixels)))

	size := int(dev.Pixels)
	if int(w.window) < len(recentWindows) {
		if w.lastClass == recentWindows[w.window].Class {
			return nil
		}
		w.lastClass = recentWindows[w.window].Class

		icon := resize.Resize(uint(size-8), uint(size-8), recentWindows[w.window].Icon, resize.Bilinear)
		draw.Draw(img, image.Rect(4, 4, size-4, size-4), icon, image.Point{0, 0}, draw.Src)
	}

	w.fg = img
	return nil
}

func (w *RecentWindowWidget) TriggerAction() {
	if xorg == nil {
		log.Println("xorg support is disabled!")
		return
	}

	if int(w.window) < len(recentWindows) {
		xorg.RequestActivation(recentWindows[w.window])
	}
}
