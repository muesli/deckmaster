package main

import (
	"image"
	"image/color"

	"github.com/muesli/streamdeck"
)

// ButtonWidget is a simple widget displaying an icon and/or label.
type ButtonWidget struct {
	BaseWidget

	icon     image.Image
	label    string
	fontsize float64
	color    color.Color
}

// NewButtonWidget returns a new ButtonWidget.
func NewButtonWidget(bw BaseWidget, opts WidgetConfig) (*ButtonWidget, error) {
	bw.setInterval(opts.Interval, 0)

	var icon, label string
	_ = ConfigValue(opts.Config["icon"], &icon)
	_ = ConfigValue(opts.Config["label"], &label)
	var fontsize float64
	_ = ConfigValue(opts.Config["fontsize"], &fontsize)
	var color color.Color
	_ = ConfigValue(opts.Config["color"], &color)

	w := &ButtonWidget{
		BaseWidget: bw,
		label:      label,
		fontsize:   fontsize,
		color:      color,
	}

	if icon != "" {
		path, err := expandPath(w.base, icon)
		if err != nil {
			return nil, err
		}
		w.icon, err = loadImage(path)
		if err != nil {
			return nil, err
		}
	}

	return w, nil
}

// Update renders the widget.
func (w *ButtonWidget) Update(dev *streamdeck.Device) error {
	size := int(dev.Pixels)
	margin := size / 18
	height := size - (margin * 2)
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	if w.label != "" {
		iconsize := int((float64(height) / 3.0) * 2.0)
		bounds := img.Bounds()

		if w.icon != nil {
			err := drawImage(img,
				w.icon,
				iconsize,
				image.Pt(-1, margin))

			if err != nil {
				return err
			}

			bounds.Min.Y += iconsize + margin
			bounds.Max.Y -= margin
		}
		if w.color == nil {
			w.color = DefaultColor
		}

		drawString(img,
			bounds,
			ttfFont,
			w.label,
			dev.DPI,
			w.fontsize,
			w.color,
			image.Pt(-1, -1))
	} else if w.icon != nil {
		err := drawImage(img,
			w.icon,
			height,
			image.Pt(-1, -1))

		if err != nil {
			return err
		}
	}

	return w.render(dev, img)
}
