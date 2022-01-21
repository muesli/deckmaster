package main

import (
	"fmt"
	"image"
	"image/color"
	"io/ioutil"
	"os"

	"github.com/flopp/go-findfont"
	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
)

var (
	ttfFont     *truetype.Font
	ttfThinFont *truetype.Font
	ttfBoldFont *truetype.Font
)

// maxPointSize returns the maximum point size we can use to fit text inside
// width and height, as well as the resulting text-width in pixels.
func maxPointSize(text string, c *freetype.Context, dpi uint, width, height int) (float64, int) {
	// never let the font size exceed the requested height
	fontsize := float64(height<<6) / float64(dpi) / (64.0 / 72.0)

	// offset initial loop iteration
	fontsize++

	// find the biggest matching font size for the requested width
	var actwidth int
	for actwidth = width + 1; actwidth > width; fontsize-- {
		c.SetFontSize(fontsize)

		textExtent, err := c.DrawString(text, freetype.Pt(0, 0))
		if err != nil {
			return 0, 0
		}

		actwidth = textExtent.X.Round()
	}

	return fontsize, actwidth
}

func fontByName(font string) *truetype.Font {
	switch font {
	case "thin":
		return ttfThinFont
	case "regular":
		return ttfFont
	case "bold":
		return ttfBoldFont
	default:
		return ttfFont
	}
}

func ftContext(img *image.RGBA, ttf *truetype.Font, dpi uint, fontsize float64) *freetype.Context {
	c := freetype.NewContext()
	c.SetDPI(float64(dpi))
	c.SetFont(ttf)
	c.SetSrc(image.NewUniform(color.RGBA{0, 0, 0, 0}))
	c.SetDst(img)
	c.SetClip(img.Bounds())
	c.SetHinting(font.HintingFull)
	c.SetFontSize(fontsize)

	return c
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
		fmt.Fprintln(os.Stderr, "Error loading font:", err)
		os.Exit(1)
	}

	ttfThinFont, err = loadFont("Roboto-Thin.ttf")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error loading font:", err)
		os.Exit(1)
	}

	ttfBoldFont, err = loadFont("Roboto-Bold.ttf")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error loading font:", err)
		os.Exit(1)
	}
}
