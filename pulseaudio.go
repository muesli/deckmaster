package main

import (
	"fmt"
	"net"
	"os"
	"path"

	"github.com/jfreymuth/pulse/proto"
	pulse "github.com/jfreymuth/pulse/proto"
)

type Source = pulse.GetSourceInfoReply

type PulseAudioClient struct {
	client *pulse.Client
	conn   net.Conn
}

// NewPulseAudioClient returns a new PulseAudioClient.
func NewPulseAudioClient() (*PulseAudioClient, error) {
	client, conn, err := pulse.Connect("")
	if err != nil {
		return nil, err
	}

	props := pulse.PropList{
		"application.name":           pulse.PropListString(path.Base(os.Args[0])),
		"application.process.id":     pulse.PropListString(fmt.Sprintf("%d", os.Getpid())),
		"application.process.binary": pulse.PropListString(os.Args[0]),
	}
	err = client.Request(&pulse.SetClientName{Props: props}, &pulse.SetClientNameReply{})

	if err != nil {
		conn.Close()
		return nil, err
	}

	return &PulseAudioClient{
		client: client,
		conn:   conn,
	}, nil
}

// Close closes the connection to pulseaudio.
func (c *PulseAudioClient) Close() error {
	return c.conn.Close()
}

// DefaultSource returns the default source.
func (c *PulseAudioClient) DefaultSource() (*Source, error) {
	var source Source
	err := c.client.Request(&proto.GetSourceInfo{SourceIndex: proto.Undefined}, &source)
	if err != nil {
		return nil, err
	}
	return &source, nil
}

// SetSourceMute set the mute state of a source.
func (c *PulseAudioClient) SetSourceMute(source *Source, mute bool) error {
	verbosef("Setting pulseaudio source %s mute to %t", source.SourceName, mute)
	err := c.client.Request(&proto.SetSourceMute{SourceIndex: source.SourceIndex, Mute: mute}, nil)
	if err != nil {
		return err
	}
	return nil
}
