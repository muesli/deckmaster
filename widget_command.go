package main

import (
	"fmt"
	"image"
	"image/color"
	"os/exec"
	"strings"

	"github.com/muesli/streamdeck"
)

// CommandWidget is a widget displaying the output of command(s).
type CommandWidget struct {
	*BaseWidget

	command string
	font    string
	color   color.Color
}

// NewCommandWidget returns a new CommandWidget.
func NewCommandWidget(bw *BaseWidget, opts WidgetConfig) *CommandWidget {
	bw.setInterval(opts.Interval, 1000)

	var command, font string
	_ = ConfigValue(opts.Config["command"], &command)
	_ = ConfigValue(opts.Config["font"], &font)
	var color color.Color
	_ = ConfigValue(opts.Config["color"], &color)

	return &CommandWidget{
		BaseWidget: bw,
		command:    command,
		font:       font,
		color:      color,
	}
}

// Update renders the widget.
func (w *CommandWidget) Update(dev *streamdeck.Device) error {
	size := int(dev.Pixels)
	margin := size / 18
	height := size - (margin * 2)
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	commands := strings.Split(w.command, ";")
	fonts := strings.Split(w.font, ";")

	if len(commands) == 0 || len(w.command) == 0 {
		return fmt.Errorf("no command(s) supplied")
	}
	for len(fonts) < len(commands) {
		fonts = append(fonts, "regular")
	}

	if w.color == nil {
		w.color = DefaultColor
	}

	for i := 0; i < len(commands); i++ {
		str, err := runCommand(commands[i])
		if err != nil {
			return err
		}
		font := fontByName(fonts[i])
		lower := margin + (height/len(commands))*i
		upper := margin + (height/len(commands))*(i+1)

		drawString(img, image.Rect(0, lower, size, upper),
			font,
			str,
			dev.DPI,
			-1,
			w.color,
			image.Pt(-1, -1))
	}

	return w.render(dev, img)
}

func runCommand(command string) (string, error) {
	output, err := exec.Command("sh", "-c", command).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(string(output), "\n"), nil
}
