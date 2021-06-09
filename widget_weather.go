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

	data    WeatherData
	color   color.Color
	flatten bool
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
	bw.setInterval(opts.Interval, 60000)

	var location, unit string
	_ = ConfigValue(opts.Config["location"], &location)
	_ = ConfigValue(opts.Config["unit"], &unit)
	var color color.Color
	_ = ConfigValue(opts.Config["color"], &color)
	var flatten bool
	_ = ConfigValue(opts.Config["flatten"], &flatten)

	if color == nil {
		color = DefaultColor
	}

	return &WeatherWidget{
		BaseWidget: bw,
		data: WeatherData{
			location: location,
			unit:     unit,
		},
		color:   color,
		flatten: flatten,
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

	var iconName string
	switch cond {
	case "mm", "mmm": // cloudy
		iconName = "cloudy"
	case "m": // partly cloudy
		iconName = "partly_cloudy"
	case "=": // fog
		iconName = "fog"
	case "///", "//", "x", "x/": // rain
		iconName = "rain"
	case "/", ".": // light rain
		iconName = "rain"
	case "**", "*/*": // heavy snow
		iconName = "snow"
	case "*", "*/": // light snow
		iconName = "snow"
	case "/!/": // thunder
		iconName = "lightning"
	case "!/", "*!*": // thunder rain
		iconName = "thunder_rain"
	// case "o": // sunny
	default:
		if time.Now().Hour() < 7 || time.Now().Hour() > 21 {
			iconName = "moon"
		} else {
			iconName = "sun"
		}
	}

	weatherIcon := weatherImage("assets/weather/" + iconName + ".png")

	bw := ButtonWidget{
		BaseWidget: w.BaseWidget,
		color:      w.color,
		icon:       weatherIcon,
		label:      temp,
		flatten:    w.flatten,
	}
	return bw.Update(dev)
}

func formatUnit(unit string) string {
	switch unit {
	case "f", "fahrenheit":
		return "&u"
	case "c", "celsius":
		return "&m"
	default:
		return ""
	}
}
