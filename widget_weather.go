package main

import (
	"fmt"
	"image"
	"image/color"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/muesli/streamdeck"
)

// WeatherWidget is a widget displaying the current weather.
type WeatherWidget struct {
	BaseWidget

	location string
	unit     string
	color    color.Color
}

// NewWeatherWidget returns a new WeatherWidget.
func NewWeatherWidget(bw BaseWidget, opts WidgetConfig) *WeatherWidget {
	bw.setInterval(opts.Interval, 300000)

	var location, unit string
	_ = ConfigValue(opts.Config["location"], &location)
	_ = ConfigValue(opts.Config["unit"], &unit)
	var color color.Color
	_ = ConfigValue(opts.Config["color"], &color)

	return &WeatherWidget{
		BaseWidget: bw,
		location:   location,
		unit:       unit,
		color:      color,
	}
}

// Update renders the widget.
func (w *WeatherWidget) Update(dev *streamdeck.Device) error {
	url := "http://wttr.in/" + w.location + "?format=%x+%t" + formatUnit(w.unit)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if strings.Contains(string(body), "Unknown location") {
		return fmt.Errorf("unknown location: %s", w.location)
	}

	wttr := strings.Split(string(body), " ")
	cond := wttr[0]
	temp := strings.Replace(wttr[1], "+", "", 1)
	if w.color == nil {
		w.color = DefaultColor
	}

	pixels := 36
	var serializedCond []uint8
	/*
		The serialized conditions are formatted the following way:
		Every entry is an unsigned 8 bit value
		 if the first bit (msd) is not set this 8 bit value resembles a series of transparent pixels
		  the number of transparent pixels is defined by the lower 7 pixels
		 if the first bit is set this bit-string resembles a series of opaque pixels
		  analog the lower 7 pixels define the number of opaque pixels
		The image is sampled as follows: top to bottom and left to right
	*/
	switch cond {
	case "mm", "mmm": // cloudy
		serializedCond = []uint8{127, 127, 52, 132, 30, 136, 27, 138, 21, 144, 19, 146, 17, 147, 17, 148, 14, 152, 11, 154, 9, 156, 7, 158, 6, 158, 6, 158, 6, 158, 6, 158, 6, 157, 8, 156, 8, 155, 10, 153, 127, 127, 76}
	case "=": // fog
		serializedCond = []uint8{127, 92, 148, 16, 149, 19, 150, 14, 150, 127, 8, 141, 23, 133, 22, 141, 26, 133, 32, 131, 33, 131, 32, 132, 23, 143, 102, 146, 17, 148, 20, 136, 28, 135, 29, 135, 22, 148, 127, 127, 4}
	case "///", "//", "x", "x/": // rain
		serializedCond = []uint8{127, 105, 135, 28, 137, 23, 142, 21, 143, 21, 144, 18, 148, 15, 150, 13, 152, 12, 152, 11, 153, 11, 153, 11, 153, 12, 152, 12, 151, 14, 149, 90, 130, 6, 129, 6, 130, 18, 131, 5, 130, 5, 131, 18, 130, 6, 130, 5, 130, 60, 129, 6, 130, 26, 130, 5, 131, 26, 130, 5, 130, 127, 103}
	case "**", "*/*": // heavy snow
		serializedCond = []uint8{89, 129, 35, 130, 34, 130, 33, 132, 33, 130, 31, 129, 2, 130, 1, 130, 29, 134, 19, 129, 2, 129, 2, 129, 5, 132, 4, 129, 2, 129, 2, 129, 9, 132, 2, 129, 4, 134, 3, 129, 2, 133, 9, 132, 1, 129, 1, 139, 1, 129, 2, 131, 12, 135, 2, 133, 2, 136, 14, 133, 1, 135, 1, 133, 17, 147, 16, 130, 1, 144, 1, 130, 17, 130, 1, 132, 1, 132, 1, 130, 21, 130, 2, 130, 3, 131, 2, 129, 21, 130, 1, 132, 1, 132, 1, 130, 18, 129, 2, 143, 3, 129, 14, 149, 17, 132, 1, 141, 14, 130, 1, 133, 2, 133, 2, 134, 1, 129, 12, 145, 1, 134, 10, 132, 2, 129, 3, 135, 3, 129, 2, 132, 9, 130, 1, 129, 2, 129, 5, 132, 4, 129, 2, 129, 2, 130, 20, 132, 30, 136, 31, 130, 34, 130, 32, 133, 33, 130, 34, 130, 34, 129, 90}
	case "/", ".": // light rain
		serializedCond = []uint8{47, 129, 34, 130, 29, 131, 2, 131, 2, 130, 25, 139, 25, 138, 26, 138, 22, 146, 19, 144, 21, 142, 3, 131, 16, 141, 2, 136, 12, 153, 10, 154, 11, 154, 13, 152, 12, 154, 9, 156, 8, 130, 2, 152, 11, 154, 10, 154, 10, 154, 10, 154, 10, 153, 12, 152, 13, 150, 15, 147, 55, 130, 5, 131, 4, 131, 18, 131, 4, 131, 5, 131, 18, 131, 4, 131, 5, 130, 24, 129, 6, 130, 26, 131, 4, 131, 25, 131, 5, 131, 26, 130, 5, 130, 83}
	case "*", "*/": // light snow
		serializedCond = []uint8{127, 105, 135, 28, 137, 23, 142, 21, 143, 21, 144, 18, 148, 15, 150, 13, 152, 12, 152, 11, 153, 11, 153, 11, 153, 12, 152, 12, 151, 14, 149, 107, 129, 16, 129, 17, 131, 14, 131, 11, 129, 5, 129, 16, 129, 4, 129, 6, 131, 26, 131, 5, 129, 26, 131, 35, 129, 127, 110}
	case "m": // partly cloudy
		serializedCond = []uint8{127, 99, 130, 29, 130, 3, 130, 4, 129, 24, 136, 1, 131, 25, 138, 22, 130, 2, 138, 22, 144, 21, 145, 19, 144, 21, 141, 3, 134, 14, 141, 1, 137, 12, 153, 10, 155, 12, 152, 13, 153, 10, 155, 9, 130, 2, 152, 11, 154, 10, 154, 10, 154, 10, 154, 10, 154, 10, 153, 12, 152, 13, 150, 127, 93}
	case "o": // sunny
		serializedCond = []uint8{125, 129, 35, 130, 27, 129, 5, 131, 5, 129, 21, 131, 3, 132, 3, 130, 21, 132, 1, 133, 1, 132, 22, 142, 22, 142, 16, 131, 2, 144, 2, 131, 11, 152, 12, 151, 14, 150, 15, 148, 15, 149, 13, 153, 9, 157, 8, 156, 10, 151, 15, 148, 15, 149, 15, 150, 13, 152, 11, 154, 16, 144, 20, 143, 21, 142, 21, 132, 2, 132, 2, 131, 21, 130, 4, 131, 4, 130, 21, 129, 6, 130, 5, 129, 28, 130, 127, 34}
	case "/!/": // thunder
		serializedCond = []uint8{127, 106, 134, 28, 137, 23, 142, 21, 144, 20, 144, 19, 147, 15, 151, 12, 152, 12, 153, 10, 154, 10, 154, 10, 154, 11, 152, 12, 152, 13, 150, 15, 147, 24, 132, 31, 137, 27, 136, 32, 131, 32, 131, 33, 130, 33, 130, 33, 130, 127, 108}
	case "!/", "*!*": // thunder rain
		serializedCond = []uint8{127, 71, 132, 30, 136, 24, 141, 21, 143, 21, 144, 19, 146, 16, 150, 13, 152, 12, 152, 11, 154, 10, 154, 10, 154, 10, 154, 10, 153, 12, 152, 13, 150, 16, 146, 23, 133, 24, 130, 5, 135, 6, 130, 14, 131, 3, 136, 5, 131, 13, 131, 8, 131, 6, 130, 15, 129, 4, 129, 3, 131, 5, 129, 2, 129, 19, 130, 3, 130, 5, 130, 21, 131, 2, 130, 5, 131, 22, 129, 3, 129, 7, 129, 127, 101}
	default:
		serializedCond = []uint8{127, 63, 129, 11, 129, 23, 129, 11, 129, 23, 129, 10, 131, 22, 130, 9, 131, 22, 130, 8, 133, 20, 131, 7, 135, 19, 132, 7, 133, 19, 133, 8, 131, 20, 134, 7, 131, 19, 135, 8, 129, 5, 129, 14, 136, 7, 129, 5, 129, 13, 138, 12, 129, 12, 140, 10, 131, 10, 141, 10, 131, 11, 139, 11, 132, 11, 137, 11, 133, 12, 135, 11, 135, 12, 134, 10, 137, 11, 133, 11, 137, 12, 132, 12, 135, 13, 131, 14, 133, 14, 131, 15, 132, 15, 130, 15, 131, 16, 130, 15, 131, 16, 129, 17, 129, 17, 129, 17, 129, 35, 129, 127, 24}
	}

	offset := 5
	index := 0
	halfMax := 128
	img := image.NewRGBA(image.Rect(0, 0, int(dev.Pixels), int(dev.Pixels)))
	for _, pix := range serializedCond {
		if int(pix) < halfMax {
			index += int(pix)
		} else {
			for i := 0; i < int(pix)-halfMax; i++ {
				img.Set(index%pixels+offset, index/pixels+offset, w.color)
				index += 1
			}
		}
	}

	drawString(img,
		img.Bounds(),
		ttfFont,
		temp,
		dev.DPI,
		14,
		w.color,
		image.Pt(-1, int(dev.Pixels)-7))

	return w.render(dev, img)
}

func formatUnit(unit string) string {
	switch unit {
	case "f", "fahrenheit":
		return "&u"
	case "c", "celcius":
		return "&u"
	default:
		return ""
	}
}
