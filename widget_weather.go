package main

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	"image/color"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/muesli/streamdeck"
)

//go:embed assets/weather
var weatherImages embed.FS

func weatherImage(name string) image.Image {
	b, err := weatherImages.ReadFile(name)
	if err != nil {
		panic(err)
	}

	icon, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		panic(err)
	}

	return icon
}

// WeatherWidget is a widget displaying the current weather.
type WeatherWidget struct {
	BaseWidget

	data  WeatherData
	color color.Color
}

// WeatherData handles fetches and parsing weather data.
type WeatherData struct {
	location string
	unit     string

	refresh time.Time

	response      string
	responseMutex sync.RWMutex
}

// Condition returns the current condition.
func (w *WeatherData) Condition() (string, error) {
	w.responseMutex.RLock()
	defer w.responseMutex.RUnlock()

	if strings.Contains(w.response, "Unknown location") {
		return "", fmt.Errorf("unknown location: %s", w.location)
	}

	wttr := strings.Split(w.response, " ")
	if len(wttr) != 2 {
		return "", fmt.Errorf("can't parse weather response: %s", w.response)
	}

	return wttr[0], nil
}

// Temperature returns the current temperature.
func (w *WeatherData) Temperature() (string, error) {
	w.responseMutex.RLock()
	defer w.responseMutex.RUnlock()

	if strings.Contains(w.response, "Unknown location") {
		return "", fmt.Errorf("unknown location: %s", w.location)
	}

	wttr := strings.Split(w.response, " ")
	if len(wttr) != 2 {
		return "", fmt.Errorf("can't parse weather response: %s", w.response)
	}

	return strings.Replace(wttr[1], "+", "", 1), nil
}

// Ready returns true when weather data is available.
func (w *WeatherData) Ready() bool {
	w.responseMutex.RLock()
	defer w.responseMutex.RUnlock()

	return len(w.response) > 0
}

// Fetch retrieves weather data when required.
func (w *WeatherData) Fetch() {
	w.responseMutex.Lock()
	defer w.responseMutex.Unlock()

	if time.Since(w.refresh) < time.Minute*15 {
		return
	}
	// fmt.Println("Refreshing weather data...")

	url := "http://wttr.in/" + w.location + "?format=%x+%t" + formatUnit(w.unit)

	resp, err := http.Get(url) //nolint:gosec
	if err != nil {
		fmt.Println("Can't fetch weather data:", err)
		return
	}
	defer resp.Body.Close() //nolint:errcheck

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Can't read weather data:", err)
		return
	}

	w.refresh = time.Now()
	w.response = string(body)
}

// NewWeatherWidget returns a new WeatherWidget.
func NewWeatherWidget(bw BaseWidget, opts WidgetConfig) *WeatherWidget {
	bw.setInterval(opts.Interval, 1000)

	var location, unit string
	_ = ConfigValue(opts.Config["location"], &location)
	_ = ConfigValue(opts.Config["unit"], &unit)
	var color color.Color
	_ = ConfigValue(opts.Config["color"], &color)

	if color == nil {
		color = DefaultColor
	}

	return &WeatherWidget{
		BaseWidget: bw,
		data: WeatherData{
			location: location,
			unit:     unit,
		},
		color: color,
	}
}

// Update renders the widget.
func (w *WeatherWidget) Update(dev *streamdeck.Device) error {
	go w.data.Fetch()
	if !w.data.Ready() {
		return nil
	}

	cond, err := w.data.Condition()
	if err != nil {
		return err
	}
	temp, err := w.data.Temperature()
	if err != nil {
		return err
	}

	var weatherIcon image.Image
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
		weatherIcon = weatherImage("assets/weather/cloudy.png")
	case "m": // partly cloudy
		weatherIcon = weatherImage("assets/weather/partly_cloudy.png")
	case "=": // fog
		weatherIcon = weatherImage("assets/weather/fog.png")
	case "///", "//", "x", "x/": // rain
		weatherIcon = weatherImage("assets/weather/rain.png")
	case "/", ".": // light rain
		weatherIcon = weatherImage("assets/weather/rain.png")
	case "**", "*/*": // heavy snow
		weatherIcon = weatherImage("assets/weather/snow.png")
	case "*", "*/": // light snow
		weatherIcon = weatherImage("assets/weather/snow.png")
	case "/!/": // thunder
		weatherIcon = weatherImage("assets/weather/lightning.png")
	case "!/", "*!*": // thunder rain
		weatherIcon = weatherImage("assets/weather/thunder_rain.png")
	// case "o": // sunny
	default:
		if time.Now().Hour() < 7 || time.Now().Hour() > 21 {
			weatherIcon = weatherImage("assets/weather/moon.png")
		} else {
			weatherIcon = weatherImage("assets/weather/sun.png")
		}
	}

	bw := ButtonWidget{
		BaseWidget: w.BaseWidget,
		color:      w.color,
		icon:       weatherIcon,
		label:      temp,
	}
	return bw.Update(dev)
}

func formatUnit(unit string) string {
	switch unit {
	case "f", "fahrenheit":
		return "&u"
	case "c", "celsius":
		return "&u"
	default:
		return ""
	}
}
