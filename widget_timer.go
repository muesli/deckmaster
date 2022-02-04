package main

import (
	"image"
	"image/color"
	"strings"
	"time"
)

// TimerWidget is a widget displaying a timer
type TimerWidget struct {
	*BaseWidget

	times []time.Duration

	formats []string
	fonts   []string
	colors  []color.Color
	frames  []image.Rectangle

	adaptive        bool
	underflow       bool
	underflowColors []color.Color
	currIndex       int

	data TimerData
}

type TimerData struct {
	startTime  time.Time
	pausedTime time.Time
}

func (d *TimerData) IsPaused() bool {
	return !d.pausedTime.IsZero()
}

func (d *TimerData) IsRunning() bool {
	return !d.IsPaused() && d.HasDeadline()
}

func (d *TimerData) HasDeadline() bool {
	return !d.startTime.IsZero()
}

func (d *TimerData) Clear() {
	d.startTime = time.Time{}
	d.pausedTime = time.Time{}
}

// NewTimerWidget returns a new TimerWidget
func NewTimerWidget(bw *BaseWidget, opts WidgetConfig) *TimerWidget {
	bw.setInterval(time.Duration(opts.Interval)*time.Millisecond, time.Second/2)

	var times []time.Duration
	var formats, fonts, frameReps []string
	var colors, underflowColors []color.Color
	var adaptive, underflow bool

	_ = ConfigValue(opts.Config["times"], &times)

	_ = ConfigValue(opts.Config["format"], &formats)
	_ = ConfigValue(opts.Config["font"], &fonts)
	_ = ConfigValue(opts.Config["color"], &colors)
	_ = ConfigValue(opts.Config["layout"], &frameReps)

	_ = ConfigValue(opts.Config["adaptive"], &adaptive)
	_ = ConfigValue(opts.Config["underflow"], &underflow)
	_ = ConfigValue(opts.Config["underflowColor"], &underflowColors)

	if len(times) == 0 {
		defaultDuration, _ := time.ParseDuration("30m")
		times = append(times, defaultDuration)
	}

	layout := NewLayout(int(bw.dev.Pixels))
	frames := layout.FormatLayout(frameReps, len(formats))

	for i := 0; i < len(formats); i++ {
		if len(fonts) < i+1 {
			fonts = append(fonts, "regular")
		}
		if len(colors) < i+1 {
			colors = append(colors, DefaultColor)
		}
		if len(underflowColors) < i+1 {
			underflowColors = append(underflowColors, DefaultColor)
		}
	}

	return &TimerWidget{
		BaseWidget:      bw,
		times:           times,
		formats:         formats,
		fonts:           fonts,
		colors:          colors,
		frames:          frames,
		adaptive:        adaptive,
		underflow:       underflow,
		underflowColors: underflowColors,
		currIndex:       0,
		data: TimerData{
			startTime:  time.Time{},
			pausedTime: time.Time{},
		},
	}
}

// Update renders the widget.
func (w *TimerWidget) Update() error {
	if w.data.IsPaused() {
		return nil
	}
	size := int(w.dev.Pixels)
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	var str string

	for i := 0; i < len(w.formats); i++ {
		var fontColor = w.colors[i]

		if !w.data.HasDeadline() {
			str = Timespan(w.times[w.currIndex]).Format(w.formats[i], w.adaptive)
		} else {
			remainingDuration := time.Until(w.data.startTime.Add(w.times[w.currIndex]))
			if remainingDuration < 0 && !w.underflow {
				str = Timespan(w.times[w.currIndex]).Format(w.formats[i], w.adaptive)
				w.data.Clear()
			} else if remainingDuration < 0 && w.underflow {
				fontColor = w.underflowColors[i]
				str = Timespan(remainingDuration*-1).Format(w.formats[i], w.adaptive)
			} else {
				str = Timespan(remainingDuration).Format(w.formats[i], w.adaptive)
			}
		}
		font := fontByName(w.fonts[i])

		drawString(img,
			w.frames[i],
			font,
			str,
			w.dev.DPI,
			-1,
			fontColor,
			image.Pt(-1, -1))
	}

	return w.render(w.dev, img)
}

type Timespan time.Duration

func (t Timespan) Format(format string, adaptive bool) string {
	formatStr := format
	tm := map[string]string{
		"%h": "03",
		"%H": "15",
		"%i": "04",
		"%s": "05",
		"%I": "4",
		"%S": "5",
	}

	z := time.Unix(0, 0).UTC()
	current := z.Add(time.Duration(t))
	foundNonZero := false
	timeStr := ""
	if adaptive {
		for i := 0; i < len(formatStr); i++ {
			if formatStr[i:i+1] == "%" && len(formatStr) > i+1 {
				format := ReplaceAll(formatStr[i:i+2], tm)
				str := strings.TrimLeft(current.Format(format), "0")
				timeStr += str
				if str != "" {
					format = ReplaceAll(formatStr[i+2:], tm)
					timeStr += current.Format(format)
					break
				}
				foundNonZero = true
				i++
			} else {
				if !foundNonZero {
					timeStr += formatStr[i : i+1]
				}
			}
		}
		if timeStr == "" {
			timeStr = "0"
		}
	} else {
		format := ReplaceAll(format, tm)
		timeStr = current.Format(format)
	}
	return timeStr
}

func ReplaceAll(str string, tm map[string]string) string {
	for k, v := range tm {
		str = strings.ReplaceAll(str, k, v)
	}
	return str
}

func (w *TimerWidget) TriggerAction(hold bool) {
	if hold {
		if w.data.IsPaused() {
			w.data.Clear()
		} else if !w.data.HasDeadline() {
			w.currIndex = (w.currIndex + 1) % len(w.times)
		}
	} else {
		if w.data.IsRunning() {
			w.data.pausedTime = time.Now()
		} else if w.data.IsPaused() && w.data.HasDeadline() {
			pausedDuration := time.Now().Sub(w.data.pausedTime)
			w.data.startTime = w.data.startTime.Add(pausedDuration)
			w.data.pausedTime = time.Time{}
		} else {
			w.data.startTime = time.Now()
		}
	}
}
