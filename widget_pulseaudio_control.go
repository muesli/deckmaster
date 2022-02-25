package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
)

const (
	regexExpression     = `index: ([0-9]+)[\s\S]*?media.name = \"(.*?)\"[\s\S]*?application.name = \"(.*?)\"`
	regexGroupClientId  = 1
	regexGroupMediaName = 2
	regexGroupAppName   = 3

	listInputSinksCommand = "pacmd list-sink-inputs"
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
	if w.showTitle {
		sinkTitle, err := getSinkTitleFor(w.appName)
		if err != nil {
			return err
		}
	
		if sinkTitle != "" {
			w.label = stripTextTo(10, sinkTitle)
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

	sinkIndex, err := getSinkIndex(w.appName)
	if err != nil {
		fmt.Fprintln(os.Stderr, "can't toggle mute for pulseaudio sink index: "+sinkIndex, err)
	}

	toggleMute(sinkIndex)
}

func getSinkIndex(appName string) (string, error) {
	output, err := exec.Command("sh", "-c", listInputSinksCommand).Output()
	if err != nil {
		return "", fmt.Errorf("can't get pulseaudio sinks. 'pacmd' missing? %s", err)
	}

	var regex = regexp.MustCompile(regexExpression)
	matches := regex.FindAllStringSubmatch(string(output), -1)

	var sinkIndex string
	for match := range matches {
		if appName == matches[match][regexGroupAppName] {
			sinkIndex = matches[match][regexGroupClientId]
		}
	}

	return sinkIndex, nil
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
		return string(runes[:10])
	}
	return text
}

func getSinkTitleFor(appName string) (string, error){
	output, err := exec.Command("sh", "-c", listInputSinksCommand).Output()

	if err != nil {
		return "", fmt.Errorf("can't get pulseaudio sinks. 'pacmd' missing? %s", err)
	}

	var regex = regexp.MustCompile(regexExpression)
	matches := regex.FindAllStringSubmatch(string(output), -1)
	for match := range matches {
		if appName == matches[match][regexGroupAppName] {
			return matches[match][regexGroupMediaName], nil
		}
	}

    return "", nil
}
