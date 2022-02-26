package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
)

const (
	regexExpression     = `index: ([0-9]+)[\s\S]*?muted: (no|yes)[\s\S]*?media.name = \"(.*?)\"[\s\S]*?application.name = \"(.*?)\"`
	regexGroupClientId  = 1
	regexGroupMuted     = 2
	regexGroupMediaName = 3
	regexGroupAppName   = 4

	listInputSinksCommand = "pacmd list-sink-inputs"
)

// PulseAudioControlWidget is a widget displaying a recently activated window.
type PulseAudioControlWidget struct {
	*ButtonWidget

	appName   string
	mode      string
	showTitle bool
}

type sinkInputData struct {
	muted bool
	title string
	index string
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
	sinkInputData, err := getSinkInputDataForApp(w.appName)
	if err != nil {
		return err
	}

	var icon string
	if sinkInputData.muted {
		icon = "assets/volume-muted.png"
	} else {
		icon = "assets/volume-high.png"
	}

	if err := w.LoadImage(icon); err != nil {
		return err
	}

	if w.showTitle {
		if sinkInputData.title != "" {
			w.label = stripTextTo(10, sinkInputData.title)
			return w.ButtonWidget.Update()
		}
	}

	w.label = stripTextTo(10, w.appName)
	return w.ButtonWidget.Update()
}

// TriggerAction gets called when a button is pressed.
func (w *PulseAudioControlWidget) TriggerAction(hold bool) {
	if w.mode != "mute" {
		fmt.Fprintln(os.Stderr, "unknown mode:", w.mode)
		return
	}

	sinkInputData, err := getSinkInputDataForApp(w.appName)

	if err != nil {
		fmt.Fprintln(os.Stderr, "can't toggle mute for pulseaudio app "+w.appName, err)
	}

	toggleMute(sinkInputData.index)
}

func toggleMute(sinkIndex string) {
	err := exec.Command("sh", "-c", "pactl set-sink-input-mute "+sinkIndex+" toggle").Run()

	if err != nil {
		fmt.Fprintln(os.Stderr, "can't toggle mute for pulseaudio sink index: "+sinkIndex, err)
	}
}

func stripTextTo(maxLength int, text string) string {
	runes := []rune(text)
	if len(runes) > maxLength {
		return string(runes[:maxLength])
	}
	return text
}

func getSinkInputDataForApp(appName string) (*sinkInputData, error) {
	sinkInputData := &sinkInputData{}
	output, err := exec.Command("sh", "-c", listInputSinksCommand).Output()
	if err != nil {
		return nil, fmt.Errorf("can't get pulseaudio sinks. 'pacmd' missing? %s", err)
	}

	var regex = regexp.MustCompile(regexExpression)
	matches := regex.FindAllStringSubmatch(string(output), -1)
	for match := range matches {
		if appName == matches[match][regexGroupAppName] {
			sinkInputData.index = matches[match][regexGroupClientId]
			sinkInputData.muted = yesOrNoToBool(matches[match][regexGroupMuted])
			sinkInputData.title = matches[match][regexGroupMediaName]
		}
	}

	return sinkInputData, nil
}

func yesOrNoToBool(yesOrNo string) bool {
	switch yesOrNo {
	case "yes":
		return true
	case "no":
		return false
	}
	fmt.Fprintln(os.Stderr, "can't convert yes|no to bool: "+yesOrNo)
	return false
}
