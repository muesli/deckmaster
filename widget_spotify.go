package main

import (
	"errors"
	"fmt"
	"github.com/nfnt/resize"
	"image"
	"image/color"
	"image/draw"
	"net/http"
	"strings"
	"time"
)

// SpotifyWidget is a widget that displays album art and current track position
type SpotifyWidget struct {
	*BaseWidget

	fontsize float64
	color    color.Color
	flatten  bool
}

// NewSpotifyWidget returns a new SpotifyWidget.
func NewSpotifyWidget(bw *BaseWidget, opts WidgetConfig) (*SpotifyWidget, error) {
	bw.setInterval(time.Duration(opts.Interval)*time.Millisecond, 0)

	var fontsize float64
	_ = ConfigValue(opts.Config["fontsize"], &fontsize)
	var color color.Color
	_ = ConfigValue(opts.Config["color"], &color)
	var flatten bool
	_ = ConfigValue(opts.Config["flatten"], &flatten)

	if color == nil {
		color = DefaultColor
	}

	w := &SpotifyWidget{
		BaseWidget: bw,
		fontsize:   fontsize,
		color:      color,
		flatten:    flatten,
	}

	return w, nil
}

// Parse a string like (00:00/00:00) and return the first set of numbers 00:00
func extractFirstTimeSection(input string) (string, error) {
	// Remove the parentheses and split the string into two parts
	parts := strings.Split(strings.Trim(input, "()"), "/")

	// Check if the input string has the correct format
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid input format")
	}

	// Return the first time section
	return parts[0], nil
}

// LoadImage loads an image from disk.
func (w *SpotifyWidget) LoadImage(path string) (image.Image, error) {
	path, err := expandPath(w.base, path)
	if err != nil {
		return nil, err
	}
	icon, err := loadImage(path)
	if err != nil {
		return nil, err
	}

	return icon, nil
}

// LoadImageFromURL loads an image from a URL
func (w *SpotifyWidget) LoadImageFromURL(URL string) (image.Image, error) {
	//Get the response bytes from the url
	response, err := http.Get(URL)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return nil, errors.New("received non 200 response code")
	}

	img, _, err := image.Decode(response.Body)
	if err != nil {
		return nil, err
	}

	return img, err
}

// Update renders the widget.
func (w *SpotifyWidget) Update() error {
	size := int(w.dev.Pixels)
	margin := size / 18
	height := size - (margin * 2)
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	status, err := runCommand("spotifycli --playbackstatus")
	if err != nil {
		icon, err := loadImage("assets/spotify/off.png")
		if err != nil {
			return err
		}
		drawImage(img, icon, size, image.Pt(0, 0))
		return w.render(w.dev, img)
	}

	position, err := runCommand("spotifycli --position")
	if err != nil {
		return err
	}

	url, err := runCommand("spotifycli --arturl")
	if err != nil {
		return err
	}

	icon, err := w.LoadImageFromURL(url)
	if err != nil {
		return err
	}

	iconsize := int((float64(height) / 3.0) * 2.0)
	bounds := img.Bounds()

	// spotifycli --position returns a string with the following format
	// (00:00/00:00), so we need to extract the first part to get the
	// current playback position
	position, err = extractFirstTimeSection(position)
	if err != nil {
		return err
	}

	if icon != nil {

		icon = resize.Resize(uint(size), uint(size), icon, resize.Bilinear)
		draw.Draw(img, img.Bounds(), icon, image.Pt(0, margin), draw.Over)

		if err != nil {
			return err
		}

		if status == string("▮▮") {
			pauseIcon, err := loadImage("assets/spotify/pause.png")
			pauseIcon = resize.Resize(uint(size), uint(size), pauseIcon, resize.Bilinear)
			draw.Draw(img, img.Bounds(), pauseIcon, image.Pt(0, margin), draw.Over)

			if err != nil {
				return err
			}

		}

		bounds.Min.Y += iconsize + margin
		bounds.Max.Y -= margin
	}

	drawString(img,
		bounds,
		ttfFont,
		position,
		w.dev.DPI,
		w.fontsize,
		w.color,
		image.Pt(-1, -1))

	return w.render(w.dev, img)
}
