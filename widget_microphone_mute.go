package main

import (
	"fmt"
	"image"
	"os"
)

// MicrophoneMuteWidget is a widget displaying if the default microphone is muted.
type MicrophoneMuteWidget struct {
	*ButtonWidget
	pulse      *PulseAudioClient
	mute       bool
	iconUnmute image.Image
	iconMute   image.Image
}

// NewMicrophoneMuteWidget returns a new MicrophoneMuteWidget.
func NewMicrophoneMuteWidget(bw *BaseWidget, opts WidgetConfig) (*MicrophoneMuteWidget, error) {
	widget, err := NewButtonWidget(bw, opts)
	if err != nil {
		return nil, err
	}

	var iconUnmutePath, iconMutePath string
	_ = ConfigValue(opts.Config["icon"], &iconUnmutePath)
	_ = ConfigValue(opts.Config["iconMute"], &iconMutePath)
	iconUnmute, err := preloadImage(widget.base, iconUnmutePath)
	if err != nil {
		return nil, err
	}
	iconMute, err := preloadImage(widget.base, iconMutePath)
	if err != nil {
		return nil, err
	}

	pulse, err := NewPulseAudioClient()
	if err != nil {
		return nil, err
	}

	source, err := pulse.DefaultSource()
	if err != nil {
		return nil, err
	}

	return &MicrophoneMuteWidget{
		ButtonWidget: widget,
		pulse:        pulse,
		mute:         source.Mute,
		iconUnmute:   iconUnmute,
		iconMute:     iconMute,
	}, nil
}

// RequiresUpdate returns true when the widget wants to be repainted.
func (w *MicrophoneMuteWidget) RequiresUpdate() bool {
	source, err := w.pulse.DefaultSource()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't set pulseaudio default source mute: %s\n", err)
		return false
	}

	return w.mute != source.Mute || w.BaseWidget.RequiresUpdate()
}

// Update renders the widget.
func (w *MicrophoneMuteWidget) Update() error {
	source, err := w.pulse.DefaultSource()
	if err != nil {
		return err
	}

	if w.mute != source.Mute {
		w.mute = source.Mute

		if w.mute {
			w.SetImage(w.iconMute)
		} else {
			w.SetImage(w.iconUnmute)
		}
	}

	return w.ButtonWidget.Update()
}

// TriggerAction gets called when a button is pressed.
func (w *MicrophoneMuteWidget) TriggerAction(hold bool) {
	source, err := w.pulse.DefaultSource()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't get pulseaudio default source: %s\n", err)
		return
	}
	err = w.pulse.SetSourceMute(source, !source.Mute)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't set pulseaudio default source mute: %s\n", err)
		return
	}
}

// Close gets called when a button is unloaded.
func (w *MicrophoneMuteWidget) Close() error {
	return w.pulse.Close()
}

func preloadImage(base string, path string) (image.Image, error) {
	path, err := expandPath(base, path)
	if err != nil {
		return nil, err
	}
	icon, err := loadImage(path)
	if err != nil {
		return nil, err
	}

	return icon, nil
}
