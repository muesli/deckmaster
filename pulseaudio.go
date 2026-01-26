package main

import (
	"strings"
	"time"

	"github.com/tvidal-net/pulseaudio"
)

const (
	SinkChanged = iota
	SourceChanged
	SinkMuteChanged
	SourceMuteChanged
)

type ChangeType uint8

type PulseAudio struct {
	client        pulseaudio.Client
	currentSink   pulseaudio.Sink
	currentSource pulseaudio.Source
	updates       chan ChangeType
}

func getSink(name string, client *pulseaudio.Client) (*pulseaudio.Sink, error) {
	sinks, err := client.Sinks()
	if err != nil {
		return nil, err
	}
	for _, sink := range sinks {
		if sink.Name == name {
			return &sink, nil
		}
	}
	return nil, &pulseaudio.Error{Cmd: "getSink", Code: 3}
}

func getSource(name string, client *pulseaudio.Client) (*pulseaudio.Source, error) {
	sources, err := client.Sources()
	if err != nil {
		return nil, err
	}
	for _, source := range sources {
		if source.Name == name {
			return &source, nil
		}
	}
	return nil, &pulseaudio.Error{Cmd: "getSource", Code: 3}
}

func NewPulseAudio() (*PulseAudio, error) {
	client, err := pulseaudio.NewClient()
	if err != nil {
		return nil, err
	}

	serverInfo, err := client.ServerInfo()
	if err != nil {
		client.Close()
		return nil, err
	}

	defaultSink, err := getSink(serverInfo.DefaultSink, client)
	if err != nil {
		client.Close()
		return nil, err
	}

	defaultSource, err := getSource(serverInfo.DefaultSource, client)
	if err != nil {
		client.Close()
		return nil, err
	}
	pulseAudio := &PulseAudio{
		*client,
		*defaultSink,
		*defaultSource,
		make(chan ChangeType),
	}
	return pulseAudio, nil
}

func (pa *PulseAudio) Updates() <-chan ChangeType {
	return pa.updates
}

func (pa *PulseAudio) Start() {
	var pulseAudioUpdates <-chan struct{}
	for {
		clientUpdates, e := pa.client.Updates()
		if e != nil {
			errorLog(e, "failed to subscribe to PulseAudio updates")
			time.Sleep(time.Second)
		} else {
			pulseAudioUpdates = clientUpdates
			break
		}
	}
	for {
		_ = <-pulseAudioUpdates
		serverInfo, e := pa.client.ServerInfo()
		if e != nil {
			errorLog(e, "failed to get PulseAudio server info")
			continue
		}

		defaultSink, e := getSink(serverInfo.DefaultSink, &pa.client)
		if e != nil {
			errorLog(e, "failed to get PulseAudio sinks")
			continue
		}
		if defaultSink.Name != pa.CurrentSinkName() {
			pa.currentSink = *defaultSink
			pa.updates <- SinkChanged
		}
		if defaultSink.Muted != pa.currentSink.Muted {
			pa.currentSink = *defaultSink
			pa.updates <- SinkMuteChanged
		}

		defaultSource, e := getSource(serverInfo.DefaultSource, &pa.client)
		if e != nil {
			errorLog(e, "failed to get PulseAudio sources")
			continue
		}
		if defaultSource.Name != pa.CurrentSourceName() {
			pa.currentSource = *defaultSource
			pa.updates <- SourceChanged
		}
		if defaultSource.Muted != pa.currentSource.Muted {
			pa.currentSource = *defaultSource
			pa.updates <- SourceMuteChanged
		}
	}
}

func (pa *PulseAudio) Muted(isSinkStream bool) bool {
	if isSinkStream {
		return pa.currentSink.Muted
	} else {
		return pa.currentSource.Muted
	}
}

func (pa *PulseAudio) ToggleMute(isSinkStream bool) error {
	if isSinkStream {
		return pa.client.SetSinkMute(!pa.currentSink.Muted, pa.CurrentSinkName())
	} else {
		return pa.client.SetSourceMute(!pa.currentSource.Muted, pa.CurrentSourceName())
	}
}

func (pa *PulseAudio) CurrentSinkName() string {
	return pa.currentSink.Name
}

func (pa *PulseAudio) SetSink(partialName string) error {
	verboseLog("currentSink: %s", pa.CurrentSinkName())
	sinks, err := pa.client.Sinks()
	if err != nil {
		return err
	}
	for _, sink := range sinks {
		sinkName := sink.Name
		if sink.Name != pa.CurrentSinkName() && strings.Contains(sinkName, partialName) {
			verboseLog("setSink \"%s\"=%s", partialName, sinkName)
			return pa.client.SetDefaultSink(sinkName)
		}
	}
	return nil
}

func (pa *PulseAudio) CurrentSourceName() string {
	return pa.currentSource.Name
}

func (pa *PulseAudio) SetSource(partialName string) error {
	verboseLog("currentSource: %s", pa.CurrentSourceName())
	sources, err := pa.client.Sources()
	if err != nil {
		return err
	}
	for _, source := range sources {
		if source.MonitorSourceName == "" {
			sourceName := source.Name
			if source.Name != pa.CurrentSourceName() && strings.Contains(sourceName, partialName) {
				verboseLog("setSource \"%s\"=%s", partialName, sourceName)
				return pa.client.SetDefaultSource(sourceName)
			}
		}
	}
	return nil
}

func (pa *PulseAudio) Close() {
	pa.client.Close()
}
