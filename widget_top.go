package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"strconv"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

// TopWidget is a widget displaying the current CPU/MEM usage as a bar.
type TopWidget struct {
	*BaseWidget

	mode      string
	color     color.Color
	fillColor color.Color

	lastValue float64
}

// NewTopWidget returns a new TopWidget.
func NewTopWidget(bw *BaseWidget, opts WidgetConfig) *TopWidget {
	bw.setInterval(time.Duration(opts.Interval)*time.Millisecond, time.Second/2)

	var mode string
	_ = ConfigValue(opts.Config["mode"], &mode)
	var color, fillColor color.Color
	_ = ConfigValue(opts.Config["color"], &color)
	_ = ConfigValue(opts.Config["fillColor"], &fillColor)

	return &TopWidget{
		BaseWidget: bw,
		mode:       mode,
		color:      color,
		fillColor:  fillColor,
	}
}

// Update renders the widget.
func (w *TopWidget) Update() error {
	var value float64
	var label string

	switch w.mode {
	case "cpu":
		cpuUsage, err := cpu.Percent(0, false)
		if err != nil {
			return fmt.Errorf("can't retrieve CPU usage: %s", err)
		}

		value = cpuUsage[0]
		label = "CPU"

	case "memory":
		memory, err := mem.VirtualMemory()
		if err != nil {
			return fmt.Errorf("can't retrieve memory usage: %s", err)
		}
		value = memory.UsedPercent
		label = "MEM"

	default:
		return fmt.Errorf("unknown widget mode: %s", w.mode)
	}

	if w.lastValue == value {
		return nil
	}
	w.lastValue = value

	if w.color == nil {
		w.color = DefaultColor
	}
	if w.fillColor == nil {
		w.fillColor = color.RGBA{166, 155, 182, 255}
	}

	size := int(w.dev.Pixels)
	margin := size / 18
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	draw.Draw(img,
		image.Rect(12, 6, size-12, size-18),
		&image.Uniform{w.color},
		image.Point{}, draw.Src)
	draw.Draw(img,
		image.Rect(13, 7, size-14, size-20),
		&image.Uniform{color.RGBA{0, 0, 0, 255}},
		image.Point{}, draw.Src)
	draw.Draw(img,
		image.Rect(14, 7+int(float64(size-26)*(1-value/100)), size-15, size-21),
		&image.Uniform{w.fillColor},
		image.Point{}, draw.Src)

	// draw percentage
	bounds := img.Bounds()
	bounds.Min.Y = 6
	bounds.Max.Y -= 18

	drawString(img,
		bounds,
		ttfFont,
		strconv.FormatInt(int64(value), 10),
		w.dev.DPI,
		13,
		w.color,
		image.Pt(-1, -1))

	// draw description
	bounds = img.Bounds()
	bounds.Min.Y = size - 16
	bounds.Max.Y -= margin

	drawString(img,
		bounds,
		ttfFont,
		"% "+label,
		w.dev.DPI,
		-1,
		w.color,
		image.Pt(-1, -1))

	return w.render(w.dev, img)
}
