package main

import (
	"fmt"
	"image"
	"image/draw"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/muesli/streamdeck"
)

var (
	SPACES = regexp.MustCompile(`\s+`)
	PATH   = strings.Split(os.Getenv("PATH"), ":")
)

type WindowWidgets struct {
	resource regexp.Regexp
	title    regexp.Regexp
	widgets  map[uint8]Widget
}

// Deck is a set of widgets.
type Deck struct {
	file       string
	background image.Image
	windows    []WindowWidgets
	overrides  map[uint8]*Widget
	widgets    map[uint8]Widget
}

// LoadDeck loads a deck configuration.
func LoadDeck(dev *streamdeck.Device, base string, deck string) (*Deck, error) {
	path, err := expandPath(base, deck)
	if err != nil {
		return nil, err
	}
	verboseLog("Loading deck: %s", path)

	dc, err := LoadConfig(path)
	if err != nil {
		return nil, err
	}

	d := Deck{
		overrides: make(map[uint8]*Widget),
		widgets:   make(map[uint8]Widget),
		file:      path,
	}
	if dc.Background != "" {
		bgPath, err := expandPath(filepath.Dir(path), dc.Background)
		if err != nil {
			return nil, err
		}
		if err := d.loadBackground(dev, bgPath); err != nil {
			return nil, err
		}
	}

	keyMap := map[uint8]KeyConfig{}
	for _, k := range dc.Keys {
		keyMap[k.Index] = k
	}

	for i := uint8(0); i < dev.Keys; i++ {
		bg := d.backgroundForKey(dev, i)

		var w Widget
		if k, found := keyMap[i]; found {
			w, err = NewWidget(dev, filepath.Dir(path), k, bg)
			if err != nil {
				return nil, err
			}
		} else {
			w = NewBaseWidget(dev, filepath.Dir(path), i, nil, nil, bg)
		}

		d.widgets[i] = w
	}

	for _, w := range dc.Windows {
		if e := d.addWindow(dev, &w); e != nil {
			return nil, e
		}
	}

	return &d, nil
}

func (deck *Deck) addWindow(dev *streamdeck.Device, w *WindowConfig) error {
	verboseLog("loading window overrides %s:%s", w.Resource, w.Title)

	resource, err := regexp.Compile(w.Resource)
	if err != nil {
		errorLogF("failed to compile regex: %s", w.Resource)
		return err
	}

	title, err := regexp.Compile(w.Title)
	if err != nil {
		errorLogF("failed to compile regex: %s", w.Title)
		return err
	}

	window := WindowWidgets{
		resource: *resource,
		title:    *title,
		widgets:  make(map[uint8]Widget),
	}
	for _, key := range w.Keys {
		if e := window.addWidget(dev, deck, key); e != nil {
			errorLogF("failed to add widget %s:%s[%d]", w.Resource, w.Title, key.Index)
			return e
		}
	}
	deck.windows = append(deck.windows, window)
	return nil
}

func (ww *WindowWidgets) addWidget(dev *streamdeck.Device, deck *Deck, key KeyConfig) error {
	bg := deck.backgroundForKey(dev, key.Index)
	widget, err := NewWidget(dev, filepath.Dir(deck.file), key, bg)
	if err != nil {
		return err
	}
	ww.widgets[key.Index] = widget
	return nil
}

func (ww *WindowWidgets) Matches(window ActiveWindow) bool {
	resource := ww.resource.MatchString(window.resource)
	title := ww.title.MatchString(window.title)
	return resource && title
}

// loads a background image.
func (deck *Deck) loadBackground(dev *streamdeck.Device, bg string) error {
	f, err := os.Open(bg)
	if err != nil {
		return err
	}
	defer f.Close() //nolint:errcheck

	background, _, err := image.Decode(f)
	if err != nil {
		return err
	}

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

	deck.background = background
	return nil
}

// returns the background image for an individual key.
func (deck *Deck) backgroundForKey(dev *streamdeck.Device, key uint8) image.Image {
	padding := int(dev.Padding)
	pixels := int(dev.Pixels)
	bg := image.NewRGBA(image.Rect(0, 0, pixels, pixels))

	if deck.background != nil {
		start := image.Point{
			X: int(key%dev.Columns) * (pixels + padding),
			Y: int(key/dev.Columns) * (pixels + padding),
		}
		draw.Draw(bg, bg.Bounds(), deck.background, start, draw.Src)
	}
	return bg
}

func (deck *Deck) WindowChanged(window ActiveWindow) {
	verboseLog("windowChanged %s:%s %s", window.resource, window.title, window.id)
	var match = false
	for _, w := range deck.windows {
		if w.Matches(window) {
			verboseLog("windowMatch: %s:%s", w.resource.String(), w.title.String())
			for i, widget := range w.widgets {
				deck.overrideWidget(i, widget)
			}
			match = true
		}
	}
	if !match {
		for key := range deck.overrides {
			deck.restoreWidget(key)
		}
	}
}

func (deck *Deck) overrideWidget(key uint8, widget Widget) {
	deck.overrides[key] = &widget
	if err := widget.Update(); err != nil {
		fatal(err)
	}
}

func (deck *Deck) restoreWidget(key uint8) {
	delete(deck.overrides, key)
	if err := deck.widgets[key].Update(); err != nil {
		fatal(err)
	}
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
		errorLogF("Keyboard emulation is disabled!")
		return
	}

	kk := strings.Split(keys, "-")
	for i, k := range kk {
		k = formatKeycodes(strings.TrimSpace(k))
		kc, e := strconv.Atoi(k)
		if e != nil {
			errorLog(e, "%s is not a valid keycode", k)
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
	errorLog(clipboard.WriteAll(text), "failed to paste from the Clipboard")

	// paste the string
	emulateKeyPress("29-47") // ctrl-v
}

// executes a dbus method.
func executeDBusMethod(config *DBusConfig) {
	if e := CallDBus(config.Object, config.Path, config.Method, config.Value); e != nil {
		errorLog(e, "DBus call failed %+v", config)
	}
}

func (deck *Deck) Widgets(yield func(Widget) bool) {
	for i, w := range deck.widgets {
		override := deck.overrides[i]
		if override == nil {
			if !yield(w) {
				return
			}
		}
	}
	for i := range deck.overrides {
		widget := deck.overrides[i]
		if !yield(*widget) {
			return
		}
	}
}

func (deck *Deck) widget(key uint8) Widget {
	widget := deck.overrides[key]
	if widget != nil {
		return *widget
	}
	return deck.widgets[key]
}

// triggerAction triggers an action.
func (deck *Deck) triggerAction(dev *streamdeck.Device, index uint8, hold bool) {
	w := deck.widget(index)
	w.TriggerAction(hold)

	var a *ActionConfig
	if hold {
		a = w.ActionHold()
	} else {
		a = w.Action()
	}

	if a == nil {
		return
	}
	if a.Deck != "" {
		newDeck, err := LoadDeck(dev, filepath.Dir(deck.file), a.Deck)
		if err != nil {
			errorLog(err, "Failed to load deck %s", a.Deck)
			return
		}
		if err := dev.Clear(); err != nil {
			fatal(err)
			return
		}

		deck = newDeck
		deck.updateWidgets()
	}
	if a.Keycode != "" {
		emulateKeyPresses(a.Keycode)
	}
	if a.Paste != "" {
		emulateClipboard(a.Paste)
	}
	if a.DBus.Method != "" {
		executeDBusMethod(&a.DBus)
	}
	if a.Exec != "" {
		errorLog(executeCommand(a.Exec), "failed to execute command")
	}
	if a.Device != "" {
		switch {
		case a.Device == "sleep":
			if err := dev.Sleep(); err != nil {
				fatal(err)
			}

		case strings.HasPrefix(a.Device, "brightness"):
			deck.adjustBrightness(dev, strings.TrimPrefix(a.Device, "brightness"))

		default:
			errorLogF("Unrecognized special action: %s", a.Device)
		}
	}
}

// updateWidgets updates/repaints all the widgets.
func (deck *Deck) updateWidgets() {
	for w := range deck.Widgets {
		if !w.RequiresUpdate() {
			continue
		}

		// fmt.Println("Repaint", w.Key())
		if err := w.Update(); err != nil {
			fatal(err)
		}
	}
}

// adjustBrightness adjusts the brightness.
func (deck *Deck) adjustBrightness(dev *streamdeck.Device, value string) {
	if len(value) == 0 {
		errorLogF("no brightness value specified")
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
		v = int64(*brightnessConfig) - v
	case '+': // brightness+[n]:
		if v == math.MinInt64 {
			v = 10
		}
		v = int64(*brightnessConfig) + v
	default:
		v = math.MinInt64
	}

	if v == math.MinInt64 {
		errorLogF("could not grok the brightness from value '%s'", value)
		return
	}

	if v < 1 {
		v = 1
	} else if v > 100 {
		v = 100
	}
	if err := dev.SetBrightness(uint8(v)); err != nil {
		fatal(err)
	}

	*brightnessConfig = uint(v)
}
