package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"io/ioutil"
	"log"
	"os"
	"strconv"

	"github.com/flopp/go-findfont"
	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"github.com/muesli/streamdeck"
	"github.com/nfnt/resize"
	"golang.org/x/image/font"
)

var (
	ttfFont     *truetype.Font
	ttfThinFont *truetype.Font
	ttfBoldFont *truetype.Font
)

type Widget interface {
	Key() uint8
	Update(dev *streamdeck.Device)
	Action() *ActionConfig
	ActionHold() *ActionConfig
	TriggerAction()
}

type BaseWidget struct {
	key        uint8
	action     *ActionConfig
	actionHold *ActionConfig
}

func (w *BaseWidget) Key() uint8 {
	return w.key
}

func (w *BaseWidget) Action() *ActionConfig {
	return w.action
}

func (w *BaseWidget) ActionHold() *ActionConfig {
	return w.actionHold
}

func (w *BaseWidget) TriggerAction() {
	// just a stub
}

func NewWidget(index uint8, id string, action *ActionConfig, actionHold *ActionConfig, config map[string]string) Widget {
	bw := BaseWidget{index, action, actionHold}

	switch id {
	case "button":
		return &ButtonWidget{
			BaseWidget: bw,
			icon:       config["icon"],
			label:      config["label"],
		}

	case "clock":
		return &ClockWidget{bw}

	case "date":
		return &DateWidget{bw}

	case "recentWindow":
		i, err := strconv.ParseUint(config["window"], 10, 64)
		if err != nil {
			log.Fatal(err)
		}
		return &RecentWindowWidget{
			BaseWidget: bw,
			window:     uint8(i),
		}

	case "top":
		return &TopWidget{
			BaseWidget: bw,
			mode:       config["mode"],
			fillColor:  config["fillColor"],
		}

	default:
		// unknown widget ID
		fmt.Println("Unknown widget with ID:", id)
	}

	return nil
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
	c := freetype.NewContext()
	c.SetDPI(float64(dev.DPI))
	c.SetFont(ttf)
	c.SetSrc(image.NewUniform(color.RGBA{0, 0, 0, 0}))
	c.SetDst(img)
	c.SetClip(img.Bounds())
	c.SetHinting(font.HintingFull)
	c.SetFontSize(fontsize)

	// find text entent
	c.SetSrc(image.NewUniform(color.RGBA{0, 0, 0, 0}))

	if pt.X < 0 {
		extent, _ := c.DrawString(text, freetype.Pt(0, 0))
		actwidth := int(float64(extent.X) / 64.0)
		xcenter := float64(bounds.Dx())/2.0 - (float64(actwidth) / 2.0)
		pt = image.Pt(int(xcenter), pt.Y)
	}
	if pt.Y < 0 {
		actheight := int(float64(fontsize) * 72.0 / float64(dev.DPI))
		ycenter := float64(bounds.Dy()/2.0) + float64(actheight)
		pt = image.Pt(pt.X, bounds.Min.Y+int(ycenter))
	}

	c.SetSrc(image.NewUniform(color.RGBA{255, 255, 255, 255}))
	if _, err := c.DrawString(text, freetype.Pt(pt.X, pt.Y)); err != nil {
		log.Fatal(err)
	}
}

func loadFont(name string) (*truetype.Font, error) {
	fontPath, err := findfont.Find(name)
	if err != nil {
		return nil, err
	}

	ttf, err := ioutil.ReadFile(fontPath)
	if err != nil {
		return nil, err
	}

	return freetype.ParseFont(ttf)
}

func init() {
	var err error
	ttfFont, err = loadFont("Roboto-Regular.ttf")
	if err != nil {
		log.Fatal(err)
	}

	ttfThinFont, err = loadFont("Roboto-Thin.ttf")
	if err != nil {
		log.Fatal(err)
	}

	ttfBoldFont, err = loadFont("Roboto-Bold.ttf")
	if err != nil {
		log.Fatal(err)
	}
}
