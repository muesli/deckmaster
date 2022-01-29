package main

import (
	"fmt"
	"image"
	"image/color"
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

	re := regexp.MustCompile(`^\d{1,2}:\d{1,2}$`)
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
	size := int(w.dev.Pixels)
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	font := fontByName(w.font)
	str := ""
	split := strings.Split(w.times[w.currIndex], ":")

	if w.startTime.IsZero() {
		// drop errors since we ensured the format of times above
		mins, _ := strconv.ParseInt(split[0], 10, 64)
		secs, _ := strconv.ParseInt(split[1], 10, 64)
		str = timerRep(mins, secs)
	} else {
		duration, _ := time.ParseDuration(split[0] + "m" + split[1] + "s")
		remaining := time.Until(w.startTime.Add(duration))
		if remaining < 0 && !w.underflow {
			str = timerRep(0, 0)
		} else {
			seconds := int64(remaining.Seconds())
			str = timerRep(seconds/60, seconds%60)
		}
	}

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

func timerRep(minutes int64, seconds int64) string {
	sec_str := fmt.Sprintf("%02d", Abs(seconds))
	min_str := fmt.Sprintf("%d", Abs(minutes))
	if minutes < 0 || seconds < 0 {
		return "-" + min_str + ":" + sec_str
	}
	return min_str + ":" + sec_str
}

func Abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}
