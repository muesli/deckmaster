package main

import (
	"fmt"
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

// TopWidget is a widget displaying the current CPU/MEM usage as a bar.
type TopWidget struct {
	BaseWidget
	mode      string
	fillColor string

	lastValue float64
}

func NewTopWidget(bw BaseWidget, opts WidgetConfig) (*TopWidget, error) {
	bw.setInterval(opts.Interval, 500)

	var mode, fillColor string
	ConfigValue(opts.Config["mode"], &mode)
	ConfigValue(opts.Config["fillColor"], &fillColor)

	return &TopWidget{
		BaseWidget: bw,
		mode:       mode,
		fillColor:  fillColor,
	}, nil
}

// Update renders the widget.
func (w *TopWidget) Update(dev *streamdeck.Device) error {
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
		return fmt.Errorf("unknown widget mode: %s", w.mode)
	}

	if w.lastValue == value {
		return nil
	}
	w.lastValue = value

	if w.fillColor == "" {
		w.fillColor = "#a69bd6"
	}
	fill, err := colorful.Hex(w.fillColor)
	if err != nil {
		return fmt.Errorf("invalid color: %s", w.fillColor)
	}

	size := int(dev.Pixels)
	margin := size / 18
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
		dev.DPI,
		13,
		image.Pt(-1, -1))

	// draw description
	bounds = img.Bounds()
	bounds.Min.Y = size - 16
	bounds.Max.Y -= margin

	drawString(img,
		bounds,
		ttfFont,
		"% "+label,
		dev.DPI,
		-1,
		image.Pt(-1, -1))

	return w.render(dev, img)
}
