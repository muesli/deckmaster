package main

import (
	"fmt"
	"image"
	"os"
	"time"
)

type CommandImageWidget struct {
	*BaseWidget
	commands []string
}

// NewCommandImageWidget returns a new CommandImageWidget.
func NewCommandImageWidget(bw *BaseWidget, opts WidgetConfig) *CommandImageWidget {
	bw.setInterval(time.Duration(opts.Interval)*time.Millisecond, time.Second)

	var commands []string
	_ = ConfigValue(opts.Config["command"], &commands)

	return &CommandImageWidget{
		BaseWidget: bw,
		commands:   commands,
	}
}

func (w *CommandImageWidget) Update() error {
	size := int(w.dev.Pixels)
	margin := size / 18
	height := size - (margin * 2)
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	str, err3 := runCommand(w.commands[0])
	if err3 != nil {
		fmt.Fprintf(os.Stderr, "Running command failed: %s\n", err3)

		return err3
	}
	icon, err := loadImage(str)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Loading Image failed: %s\n", err)
	} else {
		err2 := drawImage(img,
			icon,
			height,
			image.Pt(-1, -1))
		if err2 != nil {
			return w.render(w.dev, img)
		}
	}

	return w.render(w.dev, img)
}
