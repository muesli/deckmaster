package main

import (
	"image"
	"image/color"
	"image/draw"
	"log"
	"strconv"

	"github.com/golang/freetype"
	"github.com/muesli/streamdeck"
	"github.com/shirou/gopsutil/cpu"
)

type TopWidget struct {
	BaseWidget
}

func (w *TopWidget) Update(dev *streamdeck.Device) {
	img := image.NewRGBA(image.Rect(0, 0, 72, 72))

	cpuUsage, err := cpu.Percent(0, false)
	if err != nil {
		log.Fatal(err)
	}

	draw.Draw(img, image.Rect(12, 6, 60, 54), &image.Uniform{color.RGBA{255, 255, 255, 255}}, image.ZP, draw.Src)
	draw.Draw(img, image.Rect(13, 7, 59, 53), &image.Uniform{color.RGBA{0, 0, 0, 255}}, image.ZP, draw.Src)
	draw.Draw(img, image.Rect(14, 7+int(46*(1-cpuUsage[0]/100)), 58, 53), &image.Uniform{color.RGBA{10, 10, 240, 255}}, image.ZP, draw.Src)

	drawString(img, ttfBoldFont, strconv.FormatInt(int64(cpuUsage[0]), 10), 20, freetype.Pt(-1, -1))
	drawString(img, ttfBoldFont, "% CPU", 12, freetype.Pt(-1, img.Bounds().Dx()-4))

	err = dev.SetImage(w.key, img)
	if err != nil {
		log.Fatal(err)
	}
}
