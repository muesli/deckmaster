package main

import (
	"image"
)

const (
	StreamConfig    = "stream"
	MutedConfig     = "muted"
	MicStreamConfig = "mic"
)

type MuteWidget struct {
	*ButtonWidget

	muted    image.Image
	playback bool
}

type MuteChangedMonitor interface {
	MuteChanged(isSinkStream bool)
}

func NewMuteWidget(bw *BaseWidget, opts WidgetConfig) (*MuteWidget, error) {
	button, err := NewButtonWidget(bw, opts)
	if err != nil {
		return nil, err
	}

	var muted, stream string
	_ = ConfigValue(opts.Config[MutedConfig], &muted)
	_ = ConfigValue(opts.Config[StreamConfig], &stream)

	isPlayback := stream != MicStreamConfig
	w := &MuteWidget{
		ButtonWidget: button,
		playback:     isPlayback,
	}
	if err := w.LoadImage(&w.muted, muted); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *MuteWidget) Update() error {
	if pa.Muted(w.playback) {
		return w.Draw(w.muted)
	} else {
		return w.Draw(w.icon)
	}
}

func (w *MuteWidget) TriggerAction(hold bool) {
	if !hold {
		errorLog(pa.ToggleMute(w.playback), "failed to toggle mute")
	}
}

func (w *MuteWidget) MuteChanged(playback bool) {
	if playback == w.playback {
		errorLog(w.Update(), "failed to update widget")
	}
}
