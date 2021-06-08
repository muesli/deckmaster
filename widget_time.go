package main

import (
	"fmt"
	"image"
	"image/color"
	"strings"
	"time"

	"github.com/muesli/streamdeck"
)

// TimeWidget is a widget displaying the current time/date.
type TimeWidget struct {
	BaseWidget

	format string
	font   string
	color  color.Color
}

// NewTimeWidget returns a new TimeWidget.
func NewTimeWidget(bw BaseWidget, opts WidgetConfig) *TimeWidget {
	bw.setInterval(opts.Interval, 500)

	var format, font string
	_ = ConfigValue(opts.Config["format"], &format)
	_ = ConfigValue(opts.Config["font"], &font)
	var color color.Color
	_ = ConfigValue(opts.Config["color"], &color)

	return &TimeWidget{
		BaseWidget: bw,
		format:     format,
		font:       font,
		color:      color,
	}
}

// Update renders the widget.
func (w *TimeWidget) Update(dev *streamdeck.Device) error {
	size := int(dev.Pixels)
	margin := size / 18
	height := size - (margin * 2)
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	formats := strings.Split(w.format, ";")
	fonts := strings.Split(w.font, ";")

	if len(formats) == 0 || len(w.format) == 0 {
		return fmt.Errorf("no time format supplied")
	}
	for len(fonts) < len(formats) {
		fonts = append(fonts, "regular")
	}

	if w.color == nil {
		w.color = DefaultColor
	}

	for i := 0; i < len(formats); i++ {
		str := formatTime(time.Now(), formats[i])
		font := fontByName(fonts[i])
		lower := margin + (height/len(formats))*i
		upper := margin + (height/len(formats))*(i+1)

		drawString(img, image.Rect(0, lower, size, upper),
			font,
			str,
			dev.DPI,
			-1,
			w.color,
			image.Pt(-1, -1))
	}

	return w.render(dev, img)
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
