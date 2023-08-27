package main

import (
	"fmt"
	"image"
	"image/draw"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/godbus/dbus"
	"github.com/muesli/streamdeck"
)

// Deck is a set of widgets.
type Deck struct {
	File       string
	dev        *streamdeck.Device
	Config     DeckConfig
	Background image.Image
	Widgets    []Widget
}

// LoadDeck loads a deck configuration.
func LoadDeck(dev *streamdeck.Device, base string, deck string) (*Deck, error) {
	path, err := expandPath(base, deck)
	if err != nil {
		return nil, err
	}
	verbosef("Loading deck: %s", path)

	dc, err := LoadConfig(path)
	if err != nil {
		return nil, err
	}

	d := Deck{
		File:   path,
		dev:    dev,
		Config: dc,
	}
	err = setDeckConfig(dc, &d)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func setDeckConfig(dc DeckConfig, d *Deck) error {
	if dc.Background != "" {
		bgpath, err := expandPath(filepath.Dir(d.File), dc.Background)
		if err != nil {
			return err
		}
		if err := d.loadBackground(d.dev, bgpath); err != nil {
			return err
		}
	}
	d.Config = dc
	return LoadWidgets(d)
}

func LoadWidgets(deck *Deck) error {
	deck.Widgets = []Widget{}
	keyMap := map[uint8]KeyConfig{}
	names := map[string]bool{}
	for _, k := range deck.Config.Keys {
		if k.Name != "" {
			if names[k.Name] {
				return fmt.Errorf("duplicate widgets with the name '%s'", k.Name)
			}
			names[k.Name] = true
		}
		keyMap[k.Index] = k
	}
	for i := uint8(0); i < deck.dev.Keys; i++ {
		w, err := LoadWidget(deck, i, keyMap[i])
		if err != nil {
			return err
		}
		deck.Widgets = append(deck.Widgets, w)
	}
	return nil
}

func LoadWidget(deck *Deck, i uint8, keyConfig KeyConfig) (Widget, error) {
	var err error
	bg := deck.backgroundForKey(deck.dev, i)
	var w Widget
	if (keyConfig.Index == i) && keyConfig.Widget.ID != "" {
		w, err = NewWidget(deck.dev, filepath.Dir(deck.File), keyConfig, bg)
		if err != nil {
			return nil, err
		}
	} else {
		w = NewBaseWidget(deck.dev, filepath.Dir(deck.File), i, nil, nil, bg)
	}
	return w, nil
}

func (d *Deck) GetDevice() *streamdeck.Device {
	return d.dev
}

// loads a background image.
func (d *Deck) loadBackground(dev *streamdeck.Device, bg string) error {
	f, err := os.Open(bg)
	if err != nil {
		return err
	}
	defer f.Close() //nolint:errcheck

	background, _, err := image.Decode(f)
	if err != nil {
		return err
	}

	err = d.validateBackground(background)
	if err != nil {
		return err
	}

	d.Background = background
	return nil
}

func (deck *Deck) validateBackground(background image.Image) error {
	dev := deck.dev
	rows := int(dev.Rows)
	cols := int(dev.Columns)
	padding := int(dev.Padding)
	pixels := int(dev.Pixels)

	width := cols*pixels + (cols-1)*padding
	height := rows*pixels + (rows-1)*padding
	if background.Bounds().Dx() != width ||
		background.Bounds().Dy() != height {
		return fmt.Errorf("supplied background image has wrong dimensions, expected %dx%d pixels", width, height)
	}
	return nil
}

// returns the background image for an individual key.
func (d Deck) backgroundForKey(dev *streamdeck.Device, key uint8) image.Image {
	padding := int(dev.Padding)
	pixels := int(dev.Pixels)
	bg := image.NewRGBA(image.Rect(0, 0, pixels, pixels))

	if d.Background != nil {
		startx := int(key%dev.Columns) * (pixels + padding)
		starty := int(key/dev.Columns) * (pixels + padding)
		draw.Draw(bg, bg.Bounds(), d.Background, image.Point{startx, starty}, draw.Src)
	}

	return bg
}

func (deck *Deck) ValidateWidgetBackground(background image.Image) error {
	dev := deck.dev
	rows := int(dev.Rows)
	cols := int(dev.Columns)
	padding := int(dev.Padding)
	pixels := int(dev.Pixels)

	width := cols*pixels + (cols-1)*padding
	height := rows*pixels + (rows-1)*padding
	if background.Bounds().Dx() != width ||
		background.Bounds().Dy() != height {
		return fmt.Errorf("supplied widget image has wrong dimensions, expected %dx%d pixels", width, height)
	}
	return nil
}

// handles keypress with delay.
func emulateKeyPressWithDelay(keys string) {
	kd := strings.Split(keys, "+")
	emulateKeyPress(kd[0])
	if len(kd) == 1 {
		return
	}

	// optional delay
	if delay, err := strconv.Atoi(strings.TrimSpace(kd[1])); err == nil {
		time.Sleep(time.Duration(delay) * time.Millisecond)
	}
}

// emulates a range of key presses.
func emulateKeyPresses(keys string) {
	for _, kp := range strings.Split(keys, "/") {
		emulateKeyPressWithDelay(kp)
	}
}

// emulates a (multi-)key press.
func emulateKeyPress(keys string) {
	if keyboard == nil {
		fmt.Fprintln(os.Stderr, "Keyboard emulation is disabled!")
		return
	}

	kk := strings.Split(keys, "-")
	for i, k := range kk {
		k = formatKeycodes(strings.TrimSpace(k))
		kc, err := strconv.Atoi(k)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s is not a valid keycode: %s\n", k, err)
		}

		if i+1 < len(kk) {
			_ = keyboard.KeyDown(kc)
			defer keyboard.KeyUp(kc) //nolint:errcheck
		} else {
			_ = keyboard.KeyPress(kc)
		}
	}
}

// emulates a clipboard paste.
func emulateClipboard(text string) {
	err := clipboard.WriteAll(text)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Pasting to clipboard failed: %s\n", err)
	}

	// paste the string
	emulateKeyPress("29-47") // ctrl-v
}

// executes a dbus method.
func executeDBusMethod(object, path, method, args string) {
	call := dbusConn.Object(object, dbus.ObjectPath(path)).Call(method, 0, args)
	if call.Err != nil {

		fmt.Fprintf(os.Stderr, "dbus call failed: %s\n", call.Err)
	}
}

// executes a command.
func executeCommand(cmd string) {
	exp, err := expandPath("", cmd)
	if err == nil {
		cmd = exp
	}
	args := strings.Split(cmd, " ")

	c := exec.Command(args[0], args[1:]...) //nolint:gosec
	if *verbose {
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
	}

	if err := c.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Command failed: %s\n", err)
		return
	}
	if err := c.Wait(); err != nil {
		fmt.Fprintf(os.Stderr, "Command failed: %s\n", err)
	}
}

// triggerAction triggers an action.
func (d *Deck) triggerAction(dev *streamdeck.Device, index uint8, hold bool) {
	for _, w := range d.Widgets {
		if w.Key() != index {
			continue
		}

		var a *ActionConfig
		if hold {
			a = w.ActionHold()
		} else {
			a = w.Action()
		}

		if a == nil {
			w.TriggerAction(hold)
			continue
		}

		if a.Deck != "" {
			d, err := LoadDeck(dev, filepath.Dir(d.File), a.Deck)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Can't load deck:", err)
				return
			}
			if err := dev.Clear(); err != nil {
				fatal(err)
				return
			}

			deck = d
			deck.updateWidgets()
		}
		if a.Keycode != "" {
			emulateKeyPresses(a.Keycode)
		}
		if a.Paste != "" {
			emulateClipboard(a.Paste)
		}
		if a.DBus != nil && a.DBus.Method != "" {
			executeDBusMethod(a.DBus.Object, a.DBus.Path, a.DBus.Method, a.DBus.Value)
		}
		if a.Exec != "" {
			go executeCommand(a.Exec)
		}
		if a.Device != "" {
			switch {
			case a.Device == "sleep":
				if err := dev.Sleep(); err != nil {
					fatalf("error: %v\n", err)
				}

			case strings.HasPrefix(a.Device, "brightness"):
				d.adjustBrightness(dev, strings.TrimPrefix(a.Device, "brightness"))

			default:
				fmt.Fprintln(os.Stderr, "Unrecognized special action:", a.Device)
			}
		}
	}
}

// updateWidgets updates/repaints all the widgets.
func (d *Deck) updateWidgets() {
	updateMutex.Lock()
	for _, w := range d.Widgets {
		if !w.RequiresUpdate() {
			continue
		}

		// fmt.Println("Repaint", w.Key())
		if err := w.Update(); err != nil {
			fatalf("error: %v", err)
		}
	}
	updateMutex.Unlock()
}

func (d *Deck) replaceBackground(image image.Image) error {
	updateMutex.Lock()
	err := d.validateBackground(image)
	if err != nil {
		updateMutex.Unlock()
		return err
	}
	deck.Background = image
	for _, w := range deck.Widgets {
		w.reloadBackground()
		// fmt.Println("Repaint", w.Key())
		if err := w.Update(); err != nil {
			fatalf("error: %v", err)
		}
	}
	updateMutex.Unlock()
	return nil
}

// adjustBrightness adjusts the brightness.
func (d *Deck) adjustBrightness(dev *streamdeck.Device, value string) {
	if len(value) == 0 {
		fmt.Fprintln(os.Stderr, "No brightness value specified")
		return
	}

	v := int64(math.MinInt64)
	if len(value) > 1 {
		nv, err := strconv.ParseInt(value[1:], 10, 64)
		if err == nil {
			v = nv
		}
	}

	switch value[0] {
	case '=': // brightness=[n]:
	case '-': // brightness-[n]:
		if v == math.MinInt64 {
			v = 10
		}
		v = int64(*brightness) - v
	case '+': // brightness+[n]:
		if v == math.MinInt64 {
			v = 10
		}
		v = int64(*brightness) + v
	default:
		v = math.MinInt64
	}

	if v == math.MinInt64 {
		fmt.Fprintf(os.Stderr, "Could not grok the brightness from value '%s'\n", value)
		return
	}

	if v < 1 {
		v = 1
	} else if v > 100 {
		v = 100
	}
	if err := dev.SetBrightness(uint8(v)); err != nil {
		fatalf("error: %v\n", err)
	}

	*brightness = uint(v)
}
