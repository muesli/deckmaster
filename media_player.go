package main

import (
	"fmt"
	"os"
	"sync"

	"github.com/godbus/dbus/v5"
)

type MediaPlayer struct {
	conn      *dbus.Conn
	eventCh   chan<- interface{}
	busName   string
	ownerName string

	name   string
	busObj dbus.BusObject
	status MediaPlayerStatus

	mutex sync.RWMutex
}

func NewMediaPlayer(conn *dbus.Conn, eventCh chan<- interface{}, busName string, ownerName string) *MediaPlayer {
	return &MediaPlayer{
		conn:      conn,
		eventCh:   eventCh,
		busName:   busName,
		ownerName: ownerName,
	}
}

func (p *MediaPlayer) Initialize() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.busObj = p.conn.Object(p.busName, "/org/mpris/MediaPlayer2")
	p.name = p.busName

	var identity string
	p.busObj.Call("org.freedesktop.DBus.Properties.Get", 0, "org.mpris.MediaPlayer2", "Identity").Store(&identity)
	if len(identity) != 0 {
		p.name = identity
	}

	playbackStatus, err := p.busObj.GetProperty("org.mpris.MediaPlayer2.Player.PlaybackStatus")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error while getting playback status for player %s: %s", p.name, err)
	} else {
		if variant, ok := playbackStatus.Value().(string); ok {
			p.status.UpdatePlaybackStatus(variant)
		}
	}

	metadata, err := p.busObj.GetProperty("org.mpris.MediaPlayer2.Player.Metadata")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error while getting metadata for player %s: %s", p.name, err)
	} else {
		if variant, ok := metadata.Value().(map[string]dbus.Variant); ok {
			p.status.UpdateFromMetadata(variant)
		}
	}

	p.busObj.AddMatchSignal("org.freedesktop.DBus.Properties", "PropertiesChanged")
}

func (p *MediaPlayer) Close() {
	p.busObj.RemoveMatchSignal("org.freedesktop.DBus.Properties", "PropertiesChanged")
}

func (p *MediaPlayer) Stop() {
	p.busObj.Call("org.mpris.MediaPlayer2.Player.Stop", 0).Store()
}

func (p *MediaPlayer) Play() {
	p.busObj.Call("org.mpris.MediaPlayer2.Player.Play", 0).Store()
}

func (p *MediaPlayer) PlayPause() {
	p.busObj.Call("org.mpris.MediaPlayer2.Player.PlayPause", 0).Store()
}

func (p *MediaPlayer) Previous() {
	p.busObj.Call("org.mpris.MediaPlayer2.Player.Previous", 0).Store()
}

func (p *MediaPlayer) Next() {
	p.busObj.Call("org.mpris.MediaPlayer2.Player.Next", 0).Store()
}

func (p *MediaPlayer) Status() MediaPlayerStatus {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p.status
}

func (p *MediaPlayer) onPropertiesChanged(propertiesVariant map[string]dbus.Variant) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.status.UpdateFromPropertiesVariant(propertiesVariant)
	p.eventCh <- MediaPlayerStatusChanged{PlayerName: p.name, Status: p.status}
}
