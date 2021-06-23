package main

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	"io/ioutil"
	"log"
	"net/http"
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
	ready   bool

	response      string
	responseMutex sync.RWMutex
}

// Condition returns the current condition.
func (w *WeatherData) Condition() (string, error) {
	w.responseMutex.RLock()
	defer w.responseMutex.RUnlock()

	w.ready = false
	if strings.Contains(w.response, "Unknown location") {
		fmt.Println("unknown location:", w.location)
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

	w.ready = false
	if strings.Contains(w.response, "Unknown location") {
		fmt.Println("unknown location:", w.location)
		return "", nil
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

	return w.ready
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
		fmt.Println("can't fetch weather data:", err)
		return
	}
	defer resp.Body.Close() //nolint:errcheck

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("can't read weather data:", err)
		return
	}

	w.refresh = time.Now()
	w.response = string(body)
	w.ready = true
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
	return w.data.Ready() || w.ButtonWidget.RequiresUpdate()
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
			log.Println("using fallback icons")
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
