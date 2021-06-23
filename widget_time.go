package main

import (
	"image"
	"image/color"
	"strings"
	"time"
)

// TimeWidget is a widget displaying the current time/date.
type TimeWidget struct {
	*BaseWidget

	formats []string
	fonts   []string
	colors  []color.Color
	frames  []image.Rectangle
}

// NewTimeWidget returns a new TimeWidget.
func NewTimeWidget(bw *BaseWidget, opts WidgetConfig) *TimeWidget {
	bw.setInterval(time.Duration(opts.Interval)*time.Millisecond, time.Second/2)

	var formats, fonts, frameReps []string
	_ = ConfigValue(opts.Config["format"], &formats)
	_ = ConfigValue(opts.Config["font"], &fonts)
	_ = ConfigValue(opts.Config["layout"], &frameReps)
	var colors []color.Color
	_ = ConfigValue(opts.Config["color"], &colors)

	layout := NewLayout(int(bw.dev.Pixels))
	frames := layout.FormatLayout(frameReps, len(formats))

	for i := 0; i < len(formats); i++ {
		if len(fonts) < i+1 {
			fonts = append(fonts, "regular")
		}
		if len(colors) < i+1 {
			colors = append(colors, DefaultColor)
		}
	}

	return &TimeWidget{
		BaseWidget: bw,
		formats:    formats,
		fonts:      fonts,
		colors:     colors,
		frames:     frames,
	}
}

// Update renders the widget.
func (w *TimeWidget) Update() error {
	size := int(w.dev.Pixels)
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	for i := 0; i < len(w.formats); i++ {
		str := formatTime(time.Now(), w.formats[i])
		font := fontByName(w.fonts[i])

		drawString(img,
			w.frames[i],
			font,
			str,
			w.dev.DPI,
			-1,
			w.colors[i],
			image.Pt(-1, -1))
	}

	return w.render(w.dev, img)
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
