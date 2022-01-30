package main

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// TimerWidget is a widget displaying a timer
type TimerWidget struct {
	*BaseWidget

	times     []string
	font      string
	color     color.Color
	underflow bool
	currIndex int
	startTime time.Time
}

// NewTimerWidget returns a new TimerWidget
func NewTimerWidget(bw *BaseWidget, opts WidgetConfig) *TimerWidget {
	bw.setInterval(time.Duration(opts.Interval)*time.Millisecond, time.Second/2)

	var times []string
	_ = ConfigValue(opts.Config["times"], &times)
	var font string
	_ = ConfigValue(opts.Config["font"], &font)
	var color color.Color
	_ = ConfigValue(opts.Config["color"], &color)
	var underflow bool
	_ = ConfigValue(opts.Config["underflow"], &underflow)

	re := regexp.MustCompile(`^(\d{1,2}:){0,2}\d{1,2}$`)
	for i := 0; i < len(times); i++ {
		if !re.MatchString(times[i]) {
			times = append(times[:i], times[i+1:]...)
		}
	}
	if len(times) == 0 {
		times = append(times, "30:00")
	}
	if font == "" {
		font = "bold"
	}
	if color == nil {
		color = DefaultColor
	}

	return &TimerWidget{
		BaseWidget: bw,
		times:      times,
		font:       font,
		color:      color,
		underflow:  underflow,
		currIndex:  0,
		startTime:  time.Time{},
	}
}

// Update renders the widget.
func (w *TimerWidget) Update() error {
	split := strings.Split(w.times[w.currIndex], ":")
	seconds := int64(0)
	for i := 0; i < len(split); i++ {
		val, _ := strconv.ParseInt(split[len(split)-(i+1)], 10, 64)
		seconds += val * int64(math.Pow(60, float64(i)))
	}

	str := ""
	if w.startTime.IsZero() {
		str = timerRep(seconds)
	} else {
		duration, _ := time.ParseDuration(strconv.FormatInt(seconds, 10) + "s")
		remaining := time.Until(w.startTime.Add(duration))
		if remaining < 0 && !w.underflow {
			str = timerRep(0)
		} else {
			str = timerRep(int64(remaining.Seconds()))
		}
	}

	size := int(w.dev.Pixels)
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	font := fontByName(w.font)
	drawString(img,
		image.Rect(0, 0, size, size),
		font,
		str,
		w.dev.DPI,
		-1,
		w.color,
		image.Pt(-1, -1))

	return w.render(w.dev, img)
}

func (w *TimerWidget) TriggerAction(hold bool) {
	if w.startTime.IsZero() {
		if hold {
			w.currIndex = (w.currIndex + 1) % len(w.times)
		} else {
			w.startTime = time.Now()
		}
	} else {
		w.startTime = time.Time{}
	}
}

func timerRep(seconds int64) string {
	secs := Abs(seconds % 60)
	mins := Abs(seconds / 60 % 60)
	hrs := Abs(seconds / 60 / 60)

	str := ""
	if seconds < 0 {
		str += "-"
	}
	if hrs != 0 {
		str += fmt.Sprintf("%d", hrs) + ":" + fmt.Sprintf("%02d", mins) + ":" + fmt.Sprintf("%02d", secs)
	} else {
		if mins != 0 {
			str += fmt.Sprintf("%d", mins) + ":" + fmt.Sprintf("%02d", secs)
		} else {
			str += fmt.Sprintf("%d", secs)
		}
	}

	return str
}

func Abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}
