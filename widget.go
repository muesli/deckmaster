package main

import (
	"image"
	"image/color"
	"image/draw"
	"log"
	"os"
	"strconv"
	"sync"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"github.com/muesli/streamdeck"
	"github.com/nfnt/resize"
)

type Widget interface {
	Key() uint8
	Update(dev *streamdeck.Device) error
	Action() *ActionConfig
	ActionHold() *ActionConfig
	TriggerAction()
}

type BaseWidget struct {
	key        uint8
	action     *ActionConfig
	actionHold *ActionConfig
	background image.Image
	init       *sync.Once
}

func (w *BaseWidget) Key() uint8 {
	return w.key
}

// Action returns the associated ActionConfig.
func (w *BaseWidget) Action() *ActionConfig {
	return w.action
}

func (w *BaseWidget) ActionHold() *ActionConfig {
	return w.actionHold
}

func (w *BaseWidget) TriggerAction() {
	// just a stub
}

// Update renders the widget.
func (w *BaseWidget) Update(dev *streamdeck.Device) error {
	var err error
	w.init.Do(func() {
		err = w.render(dev, nil)
	})

	return err
}

// NewBaseWidget returns a new BaseWidget.
func NewBaseWidget(index uint8, action *ActionConfig, actionHold *ActionConfig, bg image.Image) *BaseWidget {
	return &BaseWidget{
		key:        index,
		action:     action,
		actionHold: actionHold,
		background: bg,
		init:       &sync.Once{},
	}
}

// NewWidget initializes a widget.
func NewWidget(index uint8, id string, action *ActionConfig, actionHold *ActionConfig, bg image.Image, config map[string]string) Widget {
	bw := NewBaseWidget(index, action, actionHold, bg)

	switch id {
	case "button":
		return &ButtonWidget{
			BaseWidget: *bw,
			icon:       config["icon"],
			label:      config["label"],
		}

	case "clock":
		return &TimeWidget{
			BaseWidget: *bw,
			format:     "%H;%i;%s",
			font:       "bold;regular;thin",
		}

	case "date":
		return &TimeWidget{
			BaseWidget: *bw,
			format:     "%l;%d;%M",
			font:       "regular;bold;regular",
		}

	case "time":
		return &TimeWidget{
			BaseWidget: *bw,
			format:     config["format"],
			font:       config["font"],
		}

	case "recentWindow":
		i, err := strconv.ParseUint(config["window"], 10, 64)
		if err != nil {
			log.Fatal(err)
		}
		return &RecentWindowWidget{
			BaseWidget: *bw,
			window:     uint8(i),
		}

	case "top":
		return &TopWidget{
			BaseWidget: *bw,
			mode:       config["mode"],
			fillColor:  config["fillColor"],
		}

	default:
		// unknown widget ID
		log.Println("Unknown widget with ID:", id)
	}

	return nil
}

// renders the widget including its background image.
func (w *BaseWidget) render(dev *streamdeck.Device, fg image.Image) error {
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

func drawImage(img *image.RGBA, path string, size int, pt image.Point) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

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
	draw.Draw(img, image.Rect(pt.X, pt.Y, pt.X+size, pt.Y+size), icon, image.Point{0, 0}, draw.Src)

	return nil
}

func drawString(img *image.RGBA, bounds image.Rectangle, ttf *truetype.Font, text string, fontsize float64, pt image.Point) {
	c := ftContext(img, ttf, fontsize)

	if fontsize <= 0 {
		// pick biggest available height to fit the string
		fontsize, _ = maxPointSize(text, ftContext(img, ttf, fontsize), bounds.Dx(), bounds.Dy())
		c.SetFontSize(fontsize)
	}

	if pt.X < 0 {
		// center horizontally
		extent, _ := ftContext(img, ttf, fontsize).DrawString(text, freetype.Pt(0, 0))
		actwidth := extent.X.Floor()
		xcenter := float64(bounds.Dx())/2.0 - (float64(actwidth) / 2.0)
		pt = image.Pt(bounds.Min.X+int(xcenter), pt.Y)
	}
	if pt.Y < 0 {
		// center vertically
		actheight := float64(c.PointToFixed(fontsize).Round())
		ycenter := float64(bounds.Dy()/2.0) + (float64(actheight) / 2.6)
		pt = image.Pt(pt.X, bounds.Min.Y+int(ycenter))
	}

	c.SetSrc(image.NewUniform(color.RGBA{255, 255, 255, 255}))
	if _, err := c.DrawString(text, freetype.Pt(pt.X, pt.Y)); err != nil {
		log.Fatal(err)
	}
}
