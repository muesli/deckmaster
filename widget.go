package main

import (
	"image"
	"image/color"
	"image/draw"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"github.com/muesli/streamdeck"
	"github.com/nfnt/resize"
)

// Widget is an interface implemented by all available widgets.
type Widget interface {
	Key() uint8
	RequiresUpdate() bool
	Update(dev *streamdeck.Device) error
	Action() *ActionConfig
	ActionHold() *ActionConfig
	TriggerAction()
}

// BaseWidget provides common functionality required by all widgets.
type BaseWidget struct {
	key        uint8
	action     *ActionConfig
	actionHold *ActionConfig
	background image.Image
	lastUpdate time.Time
	interval   uint
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
func (w *BaseWidget) TriggerAction() {
	// just a stub
}

// RequiresUpdate returns true when the widget wants to be repainted.
func (w *BaseWidget) RequiresUpdate() bool {
	if !w.lastUpdate.IsZero() && // initial paint done
		(w.interval == 0 || // never to be repainted
			time.Since(w.lastUpdate) < time.Duration(w.interval)*time.Millisecond) {
		return false
	}

	return true
}

// Update renders the widget.
func (w *BaseWidget) Update(dev *streamdeck.Device) error {
	return w.render(dev, nil)
}

// NewBaseWidget returns a new BaseWidget.
func NewBaseWidget(index uint8, action, actionHold *ActionConfig, bg image.Image) *BaseWidget {
	return &BaseWidget{
		key:        index,
		action:     action,
		actionHold: actionHold,
		background: bg,
	}
}

// NewWidget initializes a widget.
func NewWidget(kc KeyConfig, bg image.Image) Widget {
	bw := NewBaseWidget(kc.Index, kc.Action, kc.ActionHold, bg)
	wc := kc.Widget

	switch wc.ID {
	case "button":
		bw.setInterval(wc.Interval, 0)
		return &ButtonWidget{
			BaseWidget: *bw,
			icon:       wc.Config["icon"],
			label:      wc.Config["label"],
		}

	case "clock":
		bw.setInterval(wc.Interval, 1000)
		return &TimeWidget{
			BaseWidget: *bw,
			format:     "%H;%i;%s",
			font:       "bold;regular;thin",
		}

	case "date":
		bw.setInterval(wc.Interval, 1000)
		return &TimeWidget{
			BaseWidget: *bw,
			format:     "%l;%d;%M",
			font:       "regular;bold;regular",
		}

	case "time":
		bw.setInterval(wc.Interval, 1000)
		return &TimeWidget{
			BaseWidget: *bw,
			format:     wc.Config["format"],
			font:       wc.Config["font"],
		}

	case "recentWindow":
		i, err := strconv.ParseUint(wc.Config["window"], 10, 64)
		if err != nil {
			log.Fatal(err)
		}
		return &RecentWindowWidget{
			BaseWidget: *bw,
			window:     uint8(i),
		}

	case "top":
		bw.setInterval(wc.Interval, 500)
		return &TopWidget{
			BaseWidget: *bw,
			mode:       wc.Config["mode"],
			fillColor:  wc.Config["fillColor"],
		}

	default:
		// unknown widget ID
		log.Println("Unknown widget with ID:", wc.ID)
	}

	return nil
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
func (w *BaseWidget) setInterval(interval uint, defaultInterval uint) {
	if interval == 0 {
		interval = defaultInterval
	}

	w.interval = interval
}

func drawImage(img *image.RGBA, path string, size int, pt image.Point) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close() //nolint:errcheck

	icon, _, err := image.Decode(f)
	if err != nil {
		return err
	}

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

func drawString(img *image.RGBA, bounds image.Rectangle, ttf *truetype.Font, text string, dpi uint, fontsize float64, pt image.Point) {
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

	c.SetSrc(image.NewUniform(color.RGBA{255, 255, 255, 255}))
	if _, err := c.DrawString(text, freetype.Pt(pt.X, pt.Y)); err != nil {
		log.Fatal(err)
	}
}
