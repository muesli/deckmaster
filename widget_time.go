package main

import (
	"image"
	"log"
	"strings"
	"time"

	"github.com/muesli/streamdeck"
)

type TimeWidget struct {
	BaseWidget

	format string
	font   string
}

func formatTime(t time.Time, format string) string {
	tm := map[string]string{
		"%Y": "2006",
		"%y": "06",
		"%F": "January",
		"%M": "Jan",
		"%m": "01",
		"%l": "Monday",
		"%D": "Mon",
		"%d": "02",
		"%h": "03",
		"%H": "15",
		"%i": "04",
		"%s": "05",
		"%a": "PM",
		"%t": "MST",
	}

	for k, v := range tm {
		format = strings.ReplaceAll(format, k, v)
	}

	return t.Format(format)
}

func (w *TimeWidget) Update(dev *streamdeck.Device) {
	size := int(dev.Pixels)
	margin := size / 18
	height := size - (margin * 2)
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	formats := strings.Split(w.format, ";")
	fonts := strings.Split(w.font, ";")

	if len(formats) == 0 {
		return
	}
	for len(fonts) < len(formats) {
		fonts = append(fonts, "regular")
	}

	for i := 0; i < len(formats); i++ {
		str := formatTime(time.Now(), formats[i])
		font := fontByName(fonts[i])
		lower := margin + (height/len(formats))*i
		upper := margin + (height/len(formats))*(i+1)

		drawString(img, image.Rect(0, lower, size, upper),
			font,
			str,
			-1,
			image.Pt(-1, -1))
	}

	err := dev.SetImage(w.key, img)
	if err != nil {
		log.Fatal(err)
	}
}
