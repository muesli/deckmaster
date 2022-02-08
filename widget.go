package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"os"
	"path/filepath"
	"time"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"github.com/muesli/streamdeck"
	"github.com/nfnt/resize"
)

var (
	// DefaultColor is the standard color for text rendering.
	DefaultColor = color.RGBA{255, 255, 255, 255}
)

// Widget is an interface implemented by all available widgets.
type Widget interface {
	Key() uint8
	RequiresUpdate() bool
	Update() error
	Action() *ActionConfig
	ActionHold() *ActionConfig
	TriggerAction(hold bool)
}

// BaseWidget provides common functionality required by all widgets.
type BaseWidget struct {
	base       string
	key        uint8
	action     *ActionConfig
	actionHold *ActionConfig
	dev        *streamdeck.Device
	background image.Image
	lastUpdate time.Time
	interval   time.Duration
}

// Key returns the key a widget is mapped to.
func (w *BaseWidget) Key() uint8 {
	return w.key
}

// Action returns the associated ActionConfig.
func (w *BaseWidget) Action() *ActionConfig {
	return w.action
}

// ActionHold returns the associated ActionConfig for long presses.
func (w *BaseWidget) ActionHold() *ActionConfig {
	return w.actionHold
}

// TriggerAction gets called when a button is pressed.
func (w *BaseWidget) TriggerAction(_ bool) {
	// just a stub
}

// RequiresUpdate returns true when the widget wants to be repainted.
func (w *BaseWidget) RequiresUpdate() bool {
	if !w.lastUpdate.IsZero() && // initial paint done
		(w.interval == 0 || // never to be repainted
			time.Since(w.lastUpdate) < w.interval) {
		return false
	}

	return true
}

// Update renders the widget.
func (w *BaseWidget) Update() error {
	return w.render(w.dev, nil)
}

// NewBaseWidget returns a new BaseWidget.
func NewBaseWidget(dev *streamdeck.Device, base string, index uint8, action, actionHold *ActionConfig, bg image.Image) *BaseWidget {
	return &BaseWidget{
		base:       base,
		key:        index,
		action:     action,
		actionHold: actionHold,
		dev:        dev,
		background: bg,
	}
}

// NewWidget initializes a widget.
func NewWidget(dev *streamdeck.Device, base string, kc KeyConfig, bg image.Image) (Widget, error) {
	bw := NewBaseWidget(dev, base, kc.Index, kc.Action, kc.ActionHold, bg)

	switch kc.Widget.ID {
	case "button":
		return NewButtonWidget(bw, kc.Widget)

	case "clock":
		kc.Widget.Config = make(map[string]interface{})
		kc.Widget.Config["format"] = "%H;%i;%s"
		kc.Widget.Config["font"] = "bold;regular;thin"
		return NewTimeWidget(bw, kc.Widget), nil

	case "date":
		kc.Widget.Config = make(map[string]interface{})
		kc.Widget.Config["format"] = "%l;%d;%M"
		kc.Widget.Config["font"] = "regular;bold;regular"
		return NewTimeWidget(bw, kc.Widget), nil

	case "time":
		return NewTimeWidget(bw, kc.Widget), nil

	case "recentWindow":
		return NewRecentWindowWidget(bw, kc.Widget)

	case "top":
		return NewTopWidget(bw, kc.Widget), nil

	case "command":
		return NewCommandWidget(bw, kc.Widget), nil

	case "weather":
		return NewWeatherWidget(bw, kc.Widget)

	case "timer":
		return NewTimerWidget(bw, kc.Widget), nil
	}

	// unknown widget ID
	return nil, fmt.Errorf("Unknown widget with ID %s", kc.Widget.ID)
}

// renders the widget including its background image.
func (w *BaseWidget) render(dev *streamdeck.Device, fg image.Image) error {
	w.lastUpdate = time.Now()

	pixels := int(dev.Pixels)
	img := image.NewRGBA(image.Rect(0, 0, pixels, pixels))
	if w.background != nil {
		draw.Draw(img, img.Bounds(), w.background, image.Point{}, draw.Over)
	}
	if fg != nil {
		draw.Draw(img, img.Bounds(), fg, image.Point{}, draw.Over)
	}

	return dev.SetImage(w.key, img)
}

// change the interval a widget gets rendered in.
func (w *BaseWidget) setInterval(interval time.Duration, defaultInterval time.Duration) {
	if interval == 0 {
		interval = defaultInterval
	}

	w.interval = interval
}

func loadImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close() //nolint:errcheck

	icon, _, err := image.Decode(f)
	return icon, err
}

func loadThemeImage(theme string, img string) (image.Image, error) {
	path := filepath.Join("~", ".local", "share", "deckmaster", "themes", theme, img+".png")
	abs, err := expandPath("", path)
	if err != nil {
		return nil, err
	}
	return loadImage(abs)
}

func flattenImage(img image.Image, clr color.Color) image.Image {
	bounds := img.Bounds()
	flatten := image.NewRGBA(bounds)
	draw.Draw(flatten, flatten.Bounds(), img, image.Point{}, draw.Src)
	alphaThreshold := uint32(20000)

	for x := 0; x < bounds.Dx(); x++ {
		for y := 0; y < bounds.Dy(); y++ {
			_, _, _, alpha := flatten.At(x, y).RGBA()
			if alpha > alphaThreshold {
				flatten.Set(x, y, clr)
			} else {
				flatten.Set(x, y, color.RGBA{0, 0, 0, 0})
			}
		}
	}

	return flatten
}

func drawImage(img *image.RGBA, icon image.Image, size int, pt image.Point) error {
	if pt.X < 0 {
		xcenter := float64(img.Bounds().Dx()/2.0) - (float64(size) / 2.0)
		pt = image.Pt(int(xcenter), pt.Y)
	}
	if pt.Y < 0 {
		ycenter := float64(img.Bounds().Dy()/2.0) - (float64(size) / 2.0)
		pt = image.Pt(pt.X, int(ycenter))
	}

	icon = resize.Resize(uint(size), uint(size), icon, resize.Bilinear)
	rect := image.Rect(pt.X, pt.Y, pt.X+size, pt.Y+size)
	draw.Draw(img, rect, icon, image.Point{0, 0}, draw.Src)

	return nil
}

func drawString(img *image.RGBA, bounds image.Rectangle, ttf *truetype.Font, text string, dpi uint, fontsize float64, color color.Color, pt image.Point) {
	c := ftContext(img, ttf, dpi, fontsize)

	if fontsize <= 0 {
		// pick biggest available height to fit the string
		fontsize, _ = maxPointSize(text,
			ftContext(img, ttf, dpi, fontsize), dpi,
			bounds.Dx(), bounds.Dy())
		c.SetFontSize(fontsize)
	}

	if pt.X < 0 {
		// center horizontally
		extent, _ := ftContext(img, ttf, dpi, fontsize).DrawString(text, freetype.Pt(0, 0))
		actwidth := extent.X.Floor()
		xcenter := float64(bounds.Dx())/2.0 - (float64(actwidth) / 2.0)
		pt = image.Pt(bounds.Min.X+int(xcenter), pt.Y)
	}
	if pt.Y < 0 {
		// center vertically
		actheight := c.PointToFixed(fontsize).Round()
		ycenter := float64(bounds.Dy()/2.0) + (float64(actheight) / 2.6)
		pt = image.Pt(pt.X, bounds.Min.Y+int(ycenter))
	}

	c.SetSrc(image.NewUniform(color))
	if _, err := c.DrawString(text, freetype.Pt(pt.X, pt.Y)); err != nil {
		fmt.Fprintf(os.Stderr, "Can't render string: %s\n", err)
		return
	}
}
