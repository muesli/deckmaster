package main

import (
	"bytes"
	"io/ioutil"

	"github.com/BurntSushi/toml"
)

type DBusConfig struct {
	Object string `toml:"object,omitempty"`
	Path   string `toml:"path,omitempty"`
	Method string `toml:"method,omitempty"`
	Value  string `toml:"value,omitempty"`
}

type ActionConfig struct {
	Deck    string     `toml:"deck,omitempty"`
	Keycode string     `toml:"keycode,omitempty"`
	Exec    string     `toml:"exec,omitempty"`
	Paste   string     `toml:"paste,omitempty"`
	DBus    DBusConfig `toml:"dbus,omitempty"`
}

type WidgetConfig struct {
	ID     string            `toml:"id,omitempty"`
	Config map[string]string `toml:"config,omitempty"`
}

type KeyConfig struct {
	Index  uint8         `toml:"index"`
	Widget WidgetConfig  `toml:"widget"`
	Action *ActionConfig `toml:"action,omitempty"`
}
type Keys []KeyConfig

type DeckConfig struct {
	Keys Keys `toml:"keys"`
}

// LoadConfig loads config from filename
func LoadConfig(filename string) (DeckConfig, error) {
	config := DeckConfig{}

	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return config, err
	}

	_, err = toml.Decode(string(b), &config)
	return config, err
}

// Save writes config as json to filename
func (c DeckConfig) Save(filename string) error {
	var b bytes.Buffer
	e := toml.NewEncoder(&b)
	err := e.Encode(c)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filename, b.Bytes(), 0644)
}
