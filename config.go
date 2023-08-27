package main

import (
	"bytes"
	"errors"
	"fmt"
	"image/color"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	colorful "github.com/lucasb-eyer/go-colorful"
)

// DBusConfig describes a dbus action.
type DBusConfig struct {
	Object string `toml:"object,omitempty" json:"object,omitempty"`
	Path   string `toml:"path,omitempty" json:"path,omitempty"`
	Method string `toml:"method,omitempty" json:"method,omitempty"`
	Value  string `toml:"value,omitempty" json:"value,omitempty"`
}

// ActionConfig describes an action that can be triggered.
type ActionConfig struct {
	// The name of a deck file to load.
	Deck string `toml:"deck,omitempty" json:"deck,omitempty"`
	// A keycode to send.
	Keycode string `toml:"keycode,omitempty" json:"keycode,omitempty"`
	// A command to execute.
	Exec string `toml:"exec,omitempty" json:"exec,omitempty"`
	// A string to paste.
	Paste string `toml:"paste,omitempty" json:"paste,omitempty"`
	// A device command execute, like "sleep".
	Device string `toml:"device,omitempty" json:"device,omitempty"`
	// A DBus command to execute.
	DBus *DBusConfig `toml:"dbus,omitempty" json:"dbus,omitempty"`
}

// WidgetConfig describes configuration data for widgets.
type WidgetConfig struct {
	// The type of widget to use for the key.
	ID string `toml:"id,omitempty" json:"id,omitempty"`
	// The widget's update interval in human readable format like "1s".
	Interval uint `toml:"interval,omitempty" json:"interval,omitempty"`
	// The widget specific configuration.
	Config map[string]interface{} `toml:"config,omitempty" json:"config,omitempty"`
}

// KeyConfig holds the entire configuration for a single key.
type KeyConfig struct {
	// They key index to configure.
	Index uint8 `toml:"index" json:"index"`
	// An identifying name for the key, unique per deck.
	Name string `tomk:"name,omitempty" json:"name,omitempty"`
	// The widget configuration.
	Widget WidgetConfig `toml:"widget" json:"widget"`
	// An action to perform when the key is pressed and released.
	Action *ActionConfig `toml:"action,omitempty" json:"action,omitempty"`
	// An action to perform when the key is held.
	ActionHold *ActionConfig `toml:"action_hold,omitempty" json:"action_hold,omitempty"`
}

// Keys is a slice of keys.
type Keys []KeyConfig

// DeckConfig is the central configuration struct.
//
// swagger:model
type DeckConfig struct {
	// The deck background image.
	Background string `toml:"background,omitempty" json:"background,omitempty"`
	// A parent deck the configuration overrides.
	Parent string `toml:"parent,omitempty" json:"parent,omitempty"`
	// Configuration for individual keys
	Keys Keys `toml:"keys" json:"keys"`
}

// MergeDeckConfig merges key configuration from multiple configs.
func MergeDeckConfig(base, parent *DeckConfig) DeckConfig {
	merged := make(map[byte]KeyConfig)
	for _, config := range parent.Keys {
		merged[config.Index] = config
	}
	for _, config := range base.Keys {
		merged[config.Index] = config
	}

	keys := make(Keys, 0, len(merged))
	for _, config := range merged {
		keys = append(keys, config)
	}

	background := base.Background
	if background == "" {
		background = parent.Background
	}
	return DeckConfig{background, base.Parent, keys}
}

// LoadConfigFromFile loads a DeckConfig from a file while checking for circular
// dependencies.
func LoadConfigFromFile(base, path string, files []string) (DeckConfig, error) {
	config := DeckConfig{}

	filename, err := expandPath(base, path)
	if err != nil {
		return config, err
	}

	// check for circular dependencies
	for _, prev := range files {
		// TODO: improve error message with actual file names
		if prev == filename {
			return config, errors.New("circular reference")
		}
	}

	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return config, err
	}

	_, err = toml.Decode(string(file), &config)
	if config.Parent != "" {
		parent, err := LoadConfigFromFile(base, config.Parent, append(files, filename))
		if err != nil {
			return parent, err
		}

		merged := MergeDeckConfig(&config, &parent)
		return merged, err
	}

	return config, err
}

// LoadConfig loads config from filename.
func LoadConfig(path string) (DeckConfig, error) {
	base := filepath.Dir(path)
	filename := filepath.Base(path)

	return LoadConfigFromFile(base, filename, []string{})
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
