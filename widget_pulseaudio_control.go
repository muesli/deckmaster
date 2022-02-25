package main

import (
	"fmt"
	"image"
	"os"
	"os/exec"
	"regexp"
)

const (
	regex               = `index: ([0-9]+)[\s\S]*?media.name = \"(.*?)\"[\s\S]*?application.name = \"(.*?)\"`
	regexGroupClientId  = 1
	regexGroupMediaName = 2
	regexGroupAppName   = 3
)

// PulseAudioControlWidget is a widget displaying a recently activated window.
type PulseAudioControlWidget struct {
	*ButtonWidget

	appName   string
	mode      string
	showTitle bool
}

// NewPulseAudioControlWidget returns a new PulseAudioControlWidget.
func NewPulseAudioControlWidget(bw *BaseWidget, opts WidgetConfig) (*PulseAudioControlWidget, error) {
	var appName string
	if err := ConfigValue(opts.Config["appName"], &appName); err != nil {
		return nil, err
	}

	var mode string
	if err := ConfigValue(opts.Config["mode"], &mode); err != nil {
		return nil, err
	}

	var showTitle bool
	_ = ConfigValue(opts.Config["showTitle"], &showTitle)
	widget, err := NewButtonWidget(bw, opts)

	if err != nil {
		return nil, err
	}

	return &PulseAudioControlWidget{
		ButtonWidget: widget,
		appName:      appName,
		mode:         mode,
		showTitle:    showTitle,
	}, nil
}

// RequiresUpdate returns true when the widget wants to be repainted.
func (w *PulseAudioControlWidget) RequiresUpdate() bool {
	//TODO

	return w.BaseWidget.RequiresUpdate()
}

// Update renders the widget.
func (w *PulseAudioControlWidget) Update() error {
	img := image.NewRGBA(image.Rect(0, 0, int(w.dev.Pixels), int(w.dev.Pixels)))

	if !w.showTitle {
		var appName = w.appName

		runes := []rune(appName)
		if len(runes) > 10 {
			appName = string(runes[:10])
		}

		w.label = appName
		return w.ButtonWidget.Update()
	}

	var re = regexp.MustCompile(regex)

	output, err := exec.Command("sh", "-c", "pacmd list-sink-inputs").Output()

	if err != nil {
		return fmt.Errorf("can't get pulseaudio sinks: %s", err)
	}

	var sinkTitle = ""

	matches := re.FindAllStringSubmatch(string(output), -1)

	for match := range matches {
		if w.appName == matches[match][regexGroupAppName] {
			sinkTitle = matches[match][regexGroupMediaName]
		}
	}

	if sinkTitle != "" {

		var title = sinkTitle
		if w.showTitle {

			runes := []rune(title)
			if len(runes) > 10 {
				title = string(runes[:10])
			}
		}

		w.label = title

		return w.ButtonWidget.Update()
	}

	return w.render(w.dev, img)
}

// TriggerAction gets called when a button is pressed.
func (w *PulseAudioControlWidget) TriggerAction(hold bool) {

	if w.mode != "mute" {
		fmt.Fprintln(os.Stderr, "unknown mode:", w.mode)
		return
	}

	var re = regexp.MustCompile(regex)

	output, err := exec.Command("sh", "-c", "pacmd list-sink-inputs").Output()

	if err != nil {
		fmt.Fprintln(os.Stderr, "can't get pulseaudio sinks:", err)
		return
	}

	var sinkIndex string

	matches := re.FindAllStringSubmatch(string(output), -1)

	for match := range matches {
		if w.appName == matches[match][regexGroupAppName] {
			sinkIndex = matches[match][regexGroupClientId]
		}
	}

	output, err = exec.Command("sh", "-c", "pactl set-sink-input-mute "+sinkIndex+" toggle").Output()

	if err != nil {
		fmt.Fprintln(os.Stderr, "can't toggle mute for pulseaudio sink index: "+sinkIndex, err)
		return
	}
}
