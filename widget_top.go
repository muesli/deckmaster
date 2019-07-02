package main

import (
	"image"
	"image/color"
	"image/draw"
	"log"
	"strconv"

	"github.com/golang/freetype"
	"github.com/muesli/streamdeck"
	"github.com/shirou/gopsutil/cpu"
	"golang.org/x/image/font"
)

type TopWidget struct {
	BaseWidget
}

func (w *TopWidget) Update(dev *streamdeck.Device) {
	img := image.NewRGBA(image.Rect(0, 0, 72, 72))

	cpuUsage, err := cpu.Percent(0, false)
	if err != nil {
		log.Fatal(err)
	}

	draw.Draw(img, image.Rect(12, 6, 60, 54), &image.Uniform{color.RGBA{255, 255, 255, 255}}, image.ZP, draw.Src)
	draw.Draw(img, image.Rect(13, 7, 59, 53), &image.Uniform{color.RGBA{0, 0, 0, 255}}, image.ZP, draw.Src)
	draw.Draw(img, image.Rect(14, 7+int(46*(1-cpuUsage[0]/100)), 58, 53), &image.Uniform{color.RGBA{10, 10, 240, 255}}, image.ZP, draw.Src)

	c := freetype.NewContext()
	c.SetDPI(72)
	c.SetFont(ttfFont)
	c.SetSrc(image.NewUniform(color.RGBA{0, 0, 0, 0}))
	c.SetDst(img)
	c.SetClip(img.Bounds())
	c.SetHinting(font.HintingNone)

	c.SetFontSize(20)
	c.SetSrc(image.NewUniform(color.RGBA{0, 0, 0, 0}))
	extent, _ := c.DrawString(strconv.FormatInt(int64(cpuUsage[0]), 10), freetype.Pt(0, 0))
	actwidth := int(float64(extent.X) / 64)
	actheight := c.PointToFixed(22/2.0) / 64
	xcenter := (float64(img.Bounds().Dx()) / 2.0) - (float64(actwidth) / 2.0)
	ycenter := (float64(58) / 2.0) + (float64(actheight) / 2.0)

	c.SetSrc(image.NewUniform(color.RGBA{255, 255, 255, 255}))
	_, err = c.DrawString(strconv.FormatInt(int64(cpuUsage[0]), 10), freetype.Pt(int(xcenter), int(ycenter)))
	if err != nil {
		log.Fatal(err)
	}

	c.SetFontSize(12)
	c.SetSrc(image.NewUniform(color.RGBA{0, 0, 0, 0}))
	extent, _ = c.DrawString("% CPU", freetype.Pt(0, 0))
	actwidth = int(float64(extent.X) / 64)
	xcenter = (float64(img.Bounds().Dx()) / 2.0) - (float64(actwidth) / 2.0)

	c.SetSrc(image.NewUniform(color.RGBA{255, 255, 255, 255}))
	_, err = c.DrawString("% CPU", freetype.Pt(int(xcenter), img.Bounds().Dx()-4))
	if err != nil {
		log.Fatal(err)
	}

	err = dev.SetImage(w.key, img)
	if err != nil {
		log.Fatal(err)
	}
}

func (w *TopWidget) TriggerAction() {
}
