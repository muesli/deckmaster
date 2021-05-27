package main

import (
	"image"
	"log"
	"time"
	"strings"

	"github.com/muesli/streamdeck"
	"github.com/golang/freetype/truetype"
)

type TimeWidget struct {
	BaseWidget
	format string
	font   string
}

func mapToFont(font string) *truetype.Font {
	switch font {
	case "thin":
		return ttfThinFont
	case "regular":
		return ttfFont
	case "bold":
		return ttfBoldFont
	default:
		return ttfFont
	}
}

func mapToTimeString(format string) string {
	t := time.Now()
	switch format {
	case "yyyy":
		return t.Format("2006")
	case "yy":
		return t.Format("06")
	case "mmmm":
		return t.Format("January")
	case "mmm":
		return t.Format("Jan")
	case "mm":
		return t.Format("01")
	case "dddd":
		return t.Format("Monday")
	case "ddd":
		return t.Format("Mon")
	case "dd":
		return t.Format("02")
	case "HHT":
		return t.Format("03")
	case "HH", "hour":
		return t.Format("15")
	case "MM", "min":
		return t.Format("04")
	case "ss", "SS", "sec":
		return t.Format("05")
	case "tt":
		return t.Format("PM")
	case "Z", "ZZZ":
		return t.Format("MST")
	case "o":
		return t.Format("Z07:00")
	default:
		return t.Format(format)
	}
}

func (w *TimeWidget) Update(dev *streamdeck.Device) {
	const margin = 4
	size := int(dev.Pixels)
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	height := size - (margin * 2)

	formats := strings.Split(w.format, ";")
	fonts := strings.Split(w.font, ";")

	if len(formats) == 0 {
		return
	}
	for len(fonts) < len(formats) {
		fonts = append(fonts, "regular")
	}

	pt := (float64(height) / float64(len(formats))) * 72.0 / float64(dev.DPI)

	for i:=0; i<len(formats); i++ {
		str := mapToTimeString(formats[i])
		font := mapToFont(fonts[i])
		lower := margin + (height / len(formats)) * i
		upper := margin + (height / len(formats)) * (i + 1)

		drawString(img, image.Rect(0, lower, size, upper),
			font,
			str,
			pt,
			image.Pt(-1, -1))
	}

	err := dev.SetImage(w.key, img)
	if err != nil {
		log.Fatal(err)
	}
}
