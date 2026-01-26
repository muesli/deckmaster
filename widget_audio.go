package main

import (
	"image"
	"strings"
)

const (
	AltImageConfig   = "alt"
	MainStreamConfig = "main"
	AltStreamConfig  = "stream"
)

type AudioWidget struct {
	*ButtonWidget

	alt        image.Image
	mainStream []string
	altStream  []string
}

type AudioChangedMonitor interface {
	AudioStreamChanged(changeType ChangeType)
}

func NewAudioWidget(bw *BaseWidget, opts WidgetConfig) (*AudioWidget, error) {
	button, err := NewButtonWidget(bw, opts)
	if err != nil {
		return nil, err
	}

	var mainStreamConfig, altStreamConfig, altImageConfig string
	_ = ConfigValue(opts.Config[MainStreamConfig], &mainStreamConfig)
	_ = ConfigValue(opts.Config[AltStreamConfig], &altStreamConfig)
	w := &AudioWidget{
		ButtonWidget: button,
		mainStream:   strings.Split(mainStreamConfig, ","),
		altStream:    strings.Split(altStreamConfig, ","),
	}
	_ = ConfigValue(opts.Config[AltImageConfig], &altImageConfig)
	if err := w.LoadImage(&w.alt, altImageConfig); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *AudioWidget) MainSourceStream() string {
	if len(w.mainStream) > 0 {
		return w.mainStream[0]
	}
	return ""
}

func (w *AudioWidget) MainSinkStream() string {
	if len(w.mainStream) > 1 {
		return w.mainStream[1]
	}
	return w.MainSourceStream()
}

func (w *AudioWidget) AltSourceStream() string {
	if len(w.altStream) > 0 {
		return w.altStream[0]
	}
	return ""
}

func (w *AudioWidget) AltSinkStream() string {
	if len(w.altStream) > 1 {
		return w.altStream[1]
	}
	return w.AltSourceStream()
}

func (w *AudioWidget) IsMainStreamDefault() bool {
	sinkName := w.MainSinkStream()
	if sinkName == "" {
		return !strings.Contains(pa.CurrentSinkName(), w.AltSinkStream())
	}
	return strings.Contains(pa.CurrentSinkName(), sinkName)
}

func (w *AudioWidget) SetSinkStream(alt bool) {
	if alt {
		errorLog(pa.SetSink(w.AltSinkStream()), "failed to set PulseAudio sink stream")
	} else {
		errorLog(pa.SetSink(w.MainSinkStream()), "failed to set PulseAudio sink stream")
	}
}

func (w *AudioWidget) SetSourceStream(alt bool) {
	if alt {
		errorLog(pa.SetSource(w.AltSourceStream()), "failed to set PulseAudio source stream")
	} else {
		errorLog(pa.SetSource(w.MainSourceStream()), "failed to set PulseAudio source stream")
	}
	errorLog(w.Update(), "failed to update Widget")
}

func (w *AudioWidget) Update() error {
	if w.IsMainStreamDefault() {
		return w.Draw(w.icon)
	} else {
		return w.Draw(w.alt)
	}
}

func (w *AudioWidget) TriggerAction(hold bool) {
	if !hold {
		w.SetSinkStream(w.IsMainStreamDefault())
	}
}

func (w *AudioWidget) AudioStreamChanged(changeType ChangeType) {
	if changeType == SinkChanged {
		verboseLog("SinkChanged")
		w.SetSourceStream(!w.IsMainStreamDefault())
	} else {
		verboseLog("SourceChanged")
		errorLog(w.Update(), "failed to update Widget")
	}
}
