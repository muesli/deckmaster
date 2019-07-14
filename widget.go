package main

import (
	"image"
	"image/color"
	"image/draw"
	"io/ioutil"
	"log"
	"os"
	"strconv"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"github.com/muesli/streamdeck"
	"github.com/nfnt/resize"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
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
	TriggerAction()
}

type BaseWidget struct {
	key    uint8
	action *ActionConfig
}

func (w *BaseWidget) Key() uint8 {
	return w.key
}

func (w *BaseWidget) Action() *ActionConfig {
	return w.action
}

func (w *BaseWidget) TriggerAction() {
}

func NewWidget(index uint8, id string, action *ActionConfig, config map[string]string) Widget {
	bw := BaseWidget{index, action}

	switch id {
	case "recentWindow":
		i, err := strconv.ParseUint(config["window"], 10, 64)
		if err != nil {
			log.Fatal(err)
		}
		return &RecentWindowWidget{BaseWidget: bw, window: uint8(i)}
	case "top":
		return &TopWidget{bw}
	case "clock":
		return &ClockWidget{bw}
	case "launcher":
		return &LauncherWidget{BaseWidget: bw, launch: config["exec"], icon: config["icon"]}
	}

	return nil
}

func drawImage(img *image.RGBA, path string, size uint, x uint, y uint) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	icon, _, err := image.Decode(f)
	if err != nil {
		return err
	}

	icon = resize.Resize(size, size, icon, resize.Lanczos3)
	draw.Draw(img, image.Rect(int(x), int(y), int(x+size), int(y+size)), icon, image.Point{0, 0}, draw.Src)

	return nil
}

func drawString(img *image.RGBA, ttf *truetype.Font, text string, fontsize float64, pt fixed.Point26_6) {
	c := freetype.NewContext()
	c.SetDPI(72)
	c.SetFont(ttf)
	c.SetSrc(image.NewUniform(color.RGBA{0, 0, 0, 0}))
	c.SetDst(img)
	c.SetClip(img.Bounds())
	c.SetHinting(font.HintingNone)
	c.SetFontSize(fontsize)

	// find text entent
	c.SetSrc(image.NewUniform(color.RGBA{0, 0, 0, 0}))
	extent, _ := c.DrawString(text, freetype.Pt(0, 0))
	actwidth := int(float64(extent.X) / 64)
	actheight := c.PointToFixed(fontsize/2.0) / 64
	xcenter := (float64(img.Bounds().Dx()) / 2.0) - (float64(actwidth) / 2.0)
	ycenter := (float64(58) / 2.0) + (float64(actheight) / 2.0)

	if pt.X < 0 {
		oldy := pt.Y
		pt = freetype.Pt(int(xcenter), 0)
		pt.Y = oldy
	}
	if pt.Y < 0 {
		oldx := pt.X
		pt = freetype.Pt(0, int(ycenter))
		pt.X = oldx
	}

	c.SetSrc(image.NewUniform(color.RGBA{255, 255, 255, 255}))
	_, err := c.DrawString(text, pt)
	if err != nil {
		log.Fatal(err)
	}
}

func init() {
	ttf, err := ioutil.ReadFile("/usr/share/fonts/TTF/Roboto-Regular.ttf")
	if err != nil {
		log.Fatal(err)
	}

	ttfFont, err = freetype.ParseFont(ttf)
	if err != nil {
		log.Fatal(err)
	}

	ttf, err = ioutil.ReadFile("/usr/share/fonts/TTF/Roboto-Thin.ttf")
	if err != nil {
		log.Fatal(err)
	}

	ttfThinFont, err = freetype.ParseFont(ttf)
	if err != nil {
		log.Fatal(err)
	}

	ttf, err = ioutil.ReadFile("/usr/share/fonts/TTF/Roboto-Bold.ttf")
	if err != nil {
		log.Fatal(err)
	}

	ttfBoldFont, err = freetype.ParseFont(ttf)
	if err != nil {
		log.Fatal(err)
	}
}
