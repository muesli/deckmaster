package main

import (
	"image"
	"image/color"
	"image/draw"
	"log"
	"strconv"

	colorful "github.com/lucasb-eyer/go-colorful"
	"github.com/muesli/streamdeck"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

type TopWidget struct {
	BaseWidget
	mode      string
	fillColor string

	lastValue float64
}

func (w *TopWidget) Update(dev *streamdeck.Device) {
	var value float64
	var label string
	switch w.mode {
	case "cpu":
		cpuUsage, err := cpu.Percent(0, false)
		if err != nil {
			log.Fatal(err)
		}

		value = cpuUsage[0]
		label = "CPU"

	case "memory":
		memory, err := mem.VirtualMemory()
		if err != nil {
			log.Fatal(err)
		}
		value = memory.UsedPercent
		label = "MEM"

	default:
		panic("Unknown widget mode: " + w.mode)
	}

	if w.lastValue == value {
		return
	}
	w.lastValue = value

	fill, err := colorful.Hex(w.fillColor)
	if err != nil {
		panic("Invalid color: " + w.fillColor)
	}

	size := int(dev.Pixels)
	pt := (float64(size) / 3.0) * 66.0 / float64(dev.DPI)
	ptSmall := (float64(size) / 3.0) * 40.0 / float64(dev.DPI)

	img := image.NewRGBA(image.Rect(0, 0, size, size))

	draw.Draw(img,
		image.Rect(12, 6, size-12, size-18),
		&image.Uniform{color.RGBA{255, 255, 255, 255}},
		image.Point{}, draw.Src)
	draw.Draw(img,
		image.Rect(13, 7, size-14, size-20),
		&image.Uniform{color.RGBA{0, 0, 0, 255}},
		image.Point{}, draw.Src)
	draw.Draw(img,
		image.Rect(14, 7+int(float64(size-26)*(1-value/100)), size-15, size-21),
		&image.Uniform{fill},
		image.Point{}, draw.Src)

	// draw percentage
	bounds := img.Bounds()
	bounds.Min.Y = 6
	bounds.Max.Y -= 18

	drawString(img,
		bounds,
		ttfFont,
		strconv.FormatInt(int64(value), 10),
		pt,
		image.Pt(-1, -1))

	// draw description
	bounds = img.Bounds()
	bounds.Min.Y = size - 18

	drawString(img,
		bounds,
		ttfFont,
		"% "+label,
		ptSmall,
		image.Pt(-1, -1))

	err = dev.SetImage(w.key, img)
	if err != nil {
		log.Fatal(err)
	}
}
