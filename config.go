package main

import (
	"bytes"
	"fmt"
	"image/color"
	"io/ioutil"
	"reflect"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	colorful "github.com/lucasb-eyer/go-colorful"
)

// DBusConfig describes a dbus action.
type DBusConfig struct {
	Object string `toml:"object,omitempty"`
	Path   string `toml:"path,omitempty"`
	Method string `toml:"method,omitempty"`
	Value  string `toml:"value,omitempty"`
}

// ActionConfig describes an action that can be triggered.
type ActionConfig struct {
	Deck    string     `toml:"deck,omitempty"`
	Keycode string     `toml:"keycode,omitempty"`
	Exec    string     `toml:"exec,omitempty"`
	Paste   string     `toml:"paste,omitempty"`
	DBus    DBusConfig `toml:"dbus,omitempty"`
	Special string     `toml:"special,omitempty"`
}

// WidgetConfig describes configuration data for widgets.
type WidgetConfig struct {
	ID       string                 `toml:"id,omitempty"`
	Interval uint                   `toml:"interval,omitempty"`
	Config   map[string]interface{} `toml:"config,omitempty"`
}

// KeyConfig holds the entire configuration for a single key.
type KeyConfig struct {
	Index      uint8         `toml:"index"`
	Widget     WidgetConfig  `toml:"widget"`
	Action     *ActionConfig `toml:"action,omitempty"`
	ActionHold *ActionConfig `toml:"action_hold,omitempty"`
}

// Keys is a slice of keys.
type Keys []KeyConfig

// DeckConfig is the central configuration struct.
type DeckConfig struct {
	Background string `toml:"background,omitempty"`
	Keys       Keys   `toml:"keys"`
}

// LoadConfig loads config from filename.
func LoadConfig(filename string) (DeckConfig, error) {
	config := DeckConfig{}

	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return config, err
	}

	_, err = toml.Decode(string(b), &config)
	return config, err
}

// Save writes config as json to filename.
func (c DeckConfig) Save(filename string) error {
	var b bytes.Buffer
	e := toml.NewEncoder(&b)
	if err := e.Encode(c); err != nil {
		return err
	}

	return ioutil.WriteFile(filename, b.Bytes(), 0600)
}

// ConfigValue tries to convert an interface{} to the desired type.
func ConfigValue(v interface{}, dst interface{}) error {
	switch d := dst.(type) {
	case *string:
		switch vt := v.(type) {
		case string:
			*d = vt
		default:
			return fmt.Errorf("unhandled type %+v for string conversion", reflect.TypeOf(vt))
		}

	case *bool:
		switch vt := v.(type) {
		case bool:
			*d = vt
		case string:
			b, _ := strconv.ParseBool(vt)
			*d = b
		case int64:
			*d = vt > 0
		default:
			return fmt.Errorf("unhandled type %+v for bool conversion", reflect.TypeOf(vt))
		}

	case *int64:
		switch vt := v.(type) {
		case int64:
			*d = vt
		case float64:
			*d = int64(vt)
		case string:
			x, _ := strconv.ParseInt(vt, 0, 64)
			*d = x
		default:
			return fmt.Errorf("unhandled type %+v for uint8 conversion", reflect.TypeOf(vt))
		}

	case *float64:
		switch vt := v.(type) {
		case int64:
			*d = float64(vt)
		case float64:
			*d = vt
		case string:
			x, _ := strconv.ParseFloat(vt, 64)
			*d = x
		default:
			return fmt.Errorf("unhandled type %+v for float64 conversion", reflect.TypeOf(vt))
		}

	case *color.Color:
		switch vt := v.(type) {
		case string:
			x, _ := colorful.Hex(vt)
			*d = x
		default:
			return fmt.Errorf("unhandled type %+v for color.Color conversion", reflect.TypeOf(vt))
		}

	case *[]string:
		switch vt := v.(type) {
		case string:
			*d = strings.Split(vt, ";")
		default:
			return fmt.Errorf("unhandled type %+v for []string conversion", reflect.TypeOf(vt))
		}

	case *[]color.Color:
		switch vt := v.(type) {
		case string:
			cls := strings.Split(vt, ";")
			var clrs []color.Color
			for _, cl := range cls {
				clr, _ := colorful.Hex(cl)
				clrs = append(clrs, clr)
			}
			*d = clrs
		default:
			return fmt.Errorf("unhandled type %+v for []color.Color conversion", reflect.TypeOf(vt))
		}

	default:
		return fmt.Errorf("unhandled dst type %+v", reflect.TypeOf(dst))
	}

	return nil
}
