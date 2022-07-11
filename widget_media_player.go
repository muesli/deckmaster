package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type MediaPlayerWidget struct {
	*ButtonWidget

	mode string

	iconPlaying  string
	iconPaused   string
	iconNoPlayer string

	playerName bool

	currentPlayer *string
	currentArtURL string

	currentPlaybackStatus string
}

func NewMediaPlayerWidget(bw *BaseWidget, opts WidgetConfig) (*MediaPlayerWidget, error) {
	widget, err := NewButtonWidget(bw, opts)
	if err != nil {
		return nil, err
	}

	bw.setInterval(time.Duration(opts.Interval)*time.Millisecond, 100)

	var mode, iconPlaying, iconPaused, iconNoPlayer string
	var playerName bool
	_ = ConfigValue(opts.Config["mode"], &mode)
	_ = ConfigValue(opts.Config["icon_playing"], &iconPlaying)
	_ = ConfigValue(opts.Config["icon_paused"], &iconPaused)
	_ = ConfigValue(opts.Config["icon_no_player"], &iconNoPlayer)
	_ = ConfigValue(opts.Config["player_name"], &playerName)

	return &MediaPlayerWidget{
		ButtonWidget: widget,

		mode: mode,

		iconPlaying:  iconPlaying,
		iconPaused:   iconPaused,
		iconNoPlayer: iconNoPlayer,

		playerName: playerName,
	}, nil
}

func (w *MediaPlayerWidget) Update() error {
	fresh := true

	player := mediaPlayers.ActivePlayer()

	if w.playerName {
		if w.currentPlayer != mediaPlayers.activePlayer {
			w.currentPlayer = mediaPlayers.activePlayer
			fresh = false

			w.label = ""
			if player != nil {
				w.label = player.name
			}
		}
	}

	if w.mode == "playback status" {
		if player == nil {
			if w.currentPlaybackStatus != "No player" {
				w.currentPlaybackStatus = "No player"
				fresh = false

				if err := w.LoadImage(w.iconNoPlayer); err != nil {
					return err
				}
			}
		} else {
			status := player.Status()
			if status.status != w.currentPlaybackStatus {
				w.currentPlaybackStatus = status.status
				fresh = false

				if w.currentPlaybackStatus == "Playing" {
					if err := w.LoadImage(w.iconPlaying); err != nil {
						return err
					}
				} else {
					if err := w.LoadImage(w.iconPaused); err != nil {
						return err
					}
				}
			}
		}
	}

	if w.mode == "art" || w.mode == "" {
		if player != nil {
			url := player.Status().artURL
			if url != w.currentArtURL {
				w.currentArtURL = url
				fresh = false
				w.SetImageURL(url)
			}
		} else {
			w.SetImage(nil)
		}
	}

	if !fresh {
		return w.ButtonWidget.Update()
	}

	return nil
}

func (w *MediaPlayerWidget) SetImageURL(url string) {
	if url == "" {
		w.SetImage(nil)
	} else {
		if strings.HasPrefix(url, "file://") {
			err := w.LoadImage(strings.TrimPrefix(url, "file://"))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error while opening image: %s: %s", url, err)
			}
		} else {
			img, err := imageDownloader.Download(url)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error while downloading image: %s: %s", url, err)
			}
			w.SetImage(img)
		}
	}
}
