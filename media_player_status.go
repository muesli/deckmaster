package main

import "github.com/godbus/dbus/v5"

type MediaPlayerStatus struct {
	status string

	artist      string
	albumArtist string
	album       string
	title       string
	trackNumber int
	artURL      string
}

func (s *MediaPlayerStatus) UpdateFromPropertiesVariant(p map[string]dbus.Variant) {
	if variant, found := p["PlaybackStatus"]; found {
		if val, ok := variant.Value().(string); ok {
			s.UpdatePlaybackStatus(val)
		}
	}
	if variant, found := p["Metadata"]; found {
		if metadata, ok := variant.Value().(map[string]dbus.Variant); ok {
			s.UpdateFromMetadata(metadata)
		}
	}
}

func (s *MediaPlayerStatus) UpdatePlaybackStatus(status string) {
	s.status = status
}

func (s *MediaPlayerStatus) UpdateFromMetadata(metadata map[string]dbus.Variant) {
	s.artist = s.getMetaFirstOrEmptyString(metadata, "xesam:artist")
	s.albumArtist = s.getMetaFirstOrEmptyString(metadata, "xesam:albumArtist")
	s.album = s.getMetaOrEmptyString(metadata, "xesam:album")
	s.title = s.getMetaOrEmptyString(metadata, "xesam:title")
	s.trackNumber = s.getMetaOrZero(metadata, "xesam:trackNumber")
	s.artURL = s.getMetaOrEmptyString(metadata, "mpris:artUrl")
}

func (s *MediaPlayerStatus) getMetaOrEmptyString(metadata map[string]dbus.Variant, key string) string {
	if variant, ok := metadata[key]; ok {
		if val, ok := variant.Value().(string); ok {
			return val
		}
	}

	return ""
}

func (s *MediaPlayerStatus) getMetaOrZero(metadata map[string]dbus.Variant, key string) int {
	if variant, ok := metadata[key]; ok {
		if val, ok := variant.Value().(int32); ok {
			return int(val)
		}
	}

	return 0
}

func (s *MediaPlayerStatus) getMetaFirstOrEmptyString(metadata map[string]dbus.Variant, key string) string {
	if variant, ok := metadata[key]; ok {
		if val, ok := variant.Value().([]string); ok {
			if len(val) != 0 {
				return val[0]
			}
		}
	}

	return ""
}
