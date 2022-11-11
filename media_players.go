package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/godbus/dbus/v5"
)

type ActiveMediaPlayerChanged struct {
	PlayerName string
}

type MediaPlayerStatusChanged struct {
	PlayerName string
	Status     MediaPlayerStatus
}

type MediaPlayers struct {
	conn         *dbus.Conn
	eventCh      chan<- interface{}
	players      map[string]*MediaPlayer
	activePlayer *string
}

func NewMediaPlayers(eventCh chan<- interface{}) (*MediaPlayers, error) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return nil, err
	}
	return &MediaPlayers{
		conn:    conn,
		eventCh: eventCh,
		players: make(map[string]*MediaPlayer),
	}, nil
}

func (m *MediaPlayers) Close() error {
	return m.conn.Close()
}

func (m *MediaPlayers) Run() error {
	go func() {
		var names []string
		err := m.conn.BusObject().Call("org.freedesktop.DBus.ListNames", 0).Store(&names)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error while listing all media players: %s", err)
		}

		for _, name := range names {
			if strings.HasPrefix(name, "org.mpris.MediaPlayer2.") {
				ownerName := ""
				err = m.conn.BusObject().Call("org.freedesktop.DBus.GetNameOwner", 0, name).Store(&ownerName)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error while getting owner name for media player: %s", err)
				} else {
					m.addPlayer(name, ownerName)
				}
			}
		}

		sigCh := make(chan *dbus.Signal)

		m.conn.AddMatchSignal(dbus.WithMatchMember("NameOwnerChanged"))
		m.conn.Signal(sigCh)

		for signal := range sigCh {
			m.handleSignal(signal)
		}
	}()

	return nil
}

func (m *MediaPlayers) ActivePlayer() *MediaPlayer {
	if m.activePlayer == nil {
		return nil
	}
	return m.players[*m.activePlayer]
}

func (m *MediaPlayers) SelectPlayer(offset int) {
	if len(m.players) < 2 {
		return
	}

	var keyList []string
	current := -1
	index := 0
	for key := range m.players {
		keyList = append(keyList, key)
		if key == *m.activePlayer {
			current = index
		}
		index++
	}

	newIndex := (((current + offset) % index) + index) % index
	m.activePlayer = &keyList[newIndex]
	m.eventCh <- ActiveMediaPlayerChanged{PlayerName: m.players[*m.activePlayer].name}
}

func (m *MediaPlayers) handleSignal(signal *dbus.Signal) {
	switch signal.Name {
	case "org.freedesktop.DBus.NameOwnerChanged":
		busName := signal.Body[0].(string)
		oldOwnerName := signal.Body[1].(string)
		newOwnerName := signal.Body[2].(string)
		if !strings.HasPrefix(busName, "org.mpris.MediaPlayer2.") {
			return
		}
		if len(newOwnerName) != 0 && len(oldOwnerName) == 0 {
			m.addPlayer(busName, newOwnerName)
		} else if len(oldOwnerName) != 0 && len(newOwnerName) == 0 {
			m.removePlayer(busName, oldOwnerName)
		} else {
			m.changePlayerOwner(busName, oldOwnerName, newOwnerName)
		}
	case "org.freedesktop.DBus.Properties.PropertiesChanged":
		properties := signal.Body[1].(map[string]dbus.Variant)
		if player, ok := m.players[signal.Sender]; ok {
			player.onPropertiesChanged(properties)
		}
	default:
		verbosef("Unknown signal: %+v\n", signal)
	}
}

func (m *MediaPlayers) addPlayer(busName string, ownerName string) {
	verbosef("Adding new player %s owner %s", busName, ownerName)

	player := NewMediaPlayer(m.conn, m.eventCh, busName, ownerName)
	player.Initialize()

	m.players[ownerName] = player

	if m.activePlayer == nil {
		m.activePlayer = &ownerName
		m.eventCh <- ActiveMediaPlayerChanged{PlayerName: player.name}
	}
}

func (m *MediaPlayers) removePlayer(busName string, ownerName string) {
	verbosef("Removing player %s owner %s", busName, ownerName)

	player, ok := m.players[ownerName]

	if !ok || player.busName != busName {
		return
	}

	if *m.activePlayer == ownerName {
		m.SelectPlayer(-1)
	}

	player.Close()
	delete(m.players, ownerName)

	if *m.activePlayer == ownerName {
		m.activePlayer = nil
		m.eventCh <- ActiveMediaPlayerChanged{PlayerName: ""}
	}
}

func (m *MediaPlayers) changePlayerOwner(busName string, oldOwnerName string, newOwnerName string) {
	verbosef("Changing owner of player %s from %s to %s", busName, oldOwnerName, newOwnerName)

	player, ok := m.players[oldOwnerName]

	if !ok || player.busName != busName {
		return
	}

	m.players[newOwnerName] = player
	m.players[newOwnerName].ownerName = newOwnerName
	delete(m.players, oldOwnerName)

	if m.activePlayer != nil && *m.activePlayer == oldOwnerName {
		m.activePlayer = &newOwnerName
	}
}
