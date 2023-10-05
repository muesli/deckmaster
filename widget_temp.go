package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"strconv"
	"time"

	"github.com/shirou/gopsutil/host"
)

// TempWidget is a widget displaying temperature sensor data as a bar.
type TempWidget struct {
	*BaseWidget

	label     string
	sensorKey string
	maxTemp   float64

	color     color.Color
	fillColor color.Color

	lastValue float64
}

// NewTempWidget returns a new TempWidget.
func NewTempWidget(bw *BaseWidget, opts WidgetConfig) *TempWidget {
	bw.setInterval(time.Duration(opts.Interval)*time.Millisecond, time.Second*5)

	var label, sensorKey string
	_ = ConfigValue(opts.Config["label"], &label)
	_ = ConfigValue(opts.Config["sensorKey"], &sensorKey)
	var maxTemp float64
	_ = ConfigValue(opts.Config["maxTemp"], &maxTemp)

	var color, fillColor color.Color
	_ = ConfigValue(opts.Config["color"], &color)
	_ = ConfigValue(opts.Config["fillColor"], &fillColor)

	return &TempWidget{
		BaseWidget: bw,
		label:      label,
		sensorKey:  sensorKey,
		maxTemp:    maxTemp,
		color:      color,
		fillColor:  fillColor,
	}
}

// Update renders the widget.
func (w *TempWidget) Update() error {
	var value float64

	sensors, err := host.SensorsTemperatures()
	if err != nil {
		return fmt.Errorf("can't retrieve sensors data: %s", err)
	}

	for i := range sensors {
		if sensors[i].SensorKey == w.sensorKey {
			value = sensors[i].Temperature
			break
		}
	}

	if w.lastValue == value {
		w.lastUpdate = time.Now()
		return nil
	}
	w.lastValue = value

	if w.maxTemp == 0 {
		w.maxTemp = 100
	}
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
		image.Rect(14, 7+int(float64(size-26)*(1-value/w.maxTemp)), size-15, size-21),
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
		"°C "+w.label,
		w.dev.DPI,
		-1,
		w.color,
		image.Pt(-1, -1))

	return w.render(w.dev, img)
}
