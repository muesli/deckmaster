package main

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
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
	*ButtonWidget

	data  WeatherData
	theme string
}

// WeatherData handles fetches and parsing weather data.
type WeatherData struct {
	location string
	unit     string

	refresh time.Time
	fresh   bool

	response      string
	responseMutex sync.RWMutex
}

// Condition returns the current condition.
func (w *WeatherData) Condition() (string, error) {
	w.responseMutex.RLock()
	defer w.responseMutex.RUnlock()

	if strings.Contains(w.response, "Unknown location") {
		fmt.Fprintln(os.Stderr, "unknown location:", w.location)
		return "", nil
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
		fmt.Fprintln(os.Stderr, "unknown location:", w.location)
		return "", nil
	}

	wttr := strings.Split(w.response, " ")
	if len(wttr) != 2 {
		return "", fmt.Errorf("can't parse weather response: %s", w.response)
	}

	return strings.Replace(wttr[1], "+", "", 1), nil
}

// Fresh returns true when new weather data is available.
func (w *WeatherData) Fresh() bool {
	w.responseMutex.RLock()
	defer w.responseMutex.RUnlock()

	return w.fresh
}

// Reset marks the data as stale, so that it will be fetched again.
func (w *WeatherData) Reset() {
	w.responseMutex.Lock()
	defer w.responseMutex.Unlock()

	w.fresh = false
}

// Fetch retrieves weather data when required.
func (w *WeatherData) Fetch() {
	w.responseMutex.RLock()
	lastRefresh := w.refresh
	w.responseMutex.RUnlock()

	if time.Since(lastRefresh) < time.Minute*15 {
		return
	}
	verbosef("Refreshing weather data...")

	url := "http://wttr.in/" + w.location + "?format=%x+%t" + formatUnit(w.unit)

	resp, err := http.Get(url) //nolint:gosec
	if err != nil {
		fmt.Fprintln(os.Stderr, "can't fetch weather data:", err)
		return
	}
	defer resp.Body.Close() //nolint:errcheck

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintln(os.Stderr, "can't read weather data:", err)
		return
	}

	w.responseMutex.Lock()
	defer w.responseMutex.Unlock()

	w.refresh = time.Now()
	w.response = string(body)
	w.fresh = true
}

// NewWeatherWidget returns a new WeatherWidget.
func NewWeatherWidget(bw *BaseWidget, opts WidgetConfig) (*WeatherWidget, error) {
	var location, unit, theme string
	_ = ConfigValue(opts.Config["location"], &location)
	_ = ConfigValue(opts.Config["unit"], &unit)
	_ = ConfigValue(opts.Config["theme"], &theme)

	widget, err := NewButtonWidget(bw, opts)
	if err != nil {
		return nil, err
	}
	// this needs to be called after NewButtonWidget, otherwise its value gets
	// overwritten by it.
	bw.setInterval(time.Duration(opts.Interval)*time.Millisecond, time.Minute)

	return &WeatherWidget{
		ButtonWidget: widget,
		data: WeatherData{
			location: location,
			unit:     unit,
		},
		theme: theme,
	}, nil
}

// RequiresUpdate returns true when the widget wants to be repainted.
func (w *WeatherWidget) RequiresUpdate() bool {
	return w.data.Fresh() || w.ButtonWidget.RequiresUpdate()
}

// Update renders the widget.
func (w *WeatherWidget) Update() error {
	go w.data.Fetch()

	cond, err := w.data.Condition()
	if err != nil {
		return w.render(w.dev, nil)
	}
	temp, err := w.data.Temperature()
	if err != nil {
		return w.render(w.dev, nil)
	}

	// don't trigger updates until new weather data is available
	w.data.Reset()

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
		iconName = "light_rain"
	case "**", "*/*": // heavy snow
		iconName = "heavy_snow"
	case "*", "*/": // light snow
		iconName = "light_snow"
	case "/!/": // thunder
		iconName = "thunder"
	case "!/", "*!*": // thunder rain
		iconName = "thunder_rain"
	case "o": // sunny
		if time.Now().Hour() < 7 || time.Now().Hour() > 21 {
			iconName = "moon"
		} else {
			iconName = "sun"
		}
	default:
		return w.render(w.dev, nil)
	}

	var weatherIcon image.Image
	imagePath := filepath.Join("assets", "weather", iconName+".png")
	if w.theme != "" {
		var err error
		weatherIcon, err = loadThemeImage(w.theme, iconName)
		if err != nil {
			fmt.Fprintln(os.Stderr, "weather widget using fallback icons")
			weatherIcon = weatherImage(imagePath)
		}
	} else {
		weatherIcon = weatherImage(imagePath)
	}

	w.label = temp
	w.SetImage(weatherIcon)

	return w.ButtonWidget.Update()
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
