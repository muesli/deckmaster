package main

import (
	"fmt"
	"image"
	"image/draw"
	"log"
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
	File    string
	Widgets []Widget
}

// LoadDeck loads a deck configuration.
func LoadDeck(base string, deck string, dev *streamdeck.Device) (*Deck, error) {
	if !filepath.IsAbs(deck) {
		deck = filepath.Join(base, deck)
	}
	abs, err := filepath.Abs(deck)
	if err != nil {
		return nil, err
	}
	log.Println("Loading deck:", abs)

	dc, err := LoadConfig(abs)
	if err != nil {
		return nil, err
	}

	d := Deck{
		File: abs,
	}

	padding := int(dev.Padding)
	pixels := int(dev.Pixels)
	rows := int(dev.Rows)
	cols := int(dev.Columns)
	var background image.Image
	if dc.Background.Image != "" {
		f, err := os.Open(dc.Background.Image)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		background, _, err = image.Decode(f)
		if err != nil {
			return nil, err
		}
		width := cols*pixels + (cols-1)*padding
		height := rows*pixels + (rows-1)*padding
		if background.Bounds().Dx() != width ||
			background.Bounds().Dy() != height {
			return nil, fmt.Errorf("supplied background image has wrong dimensions, expected %dx%d pixels", width, height)
		}
	}

	keyMap := map[uint8]KeyConfig{}
	for _, k := range dc.Keys {
		keyMap[k.Index] = k
	}

	for i := 0; i < cols*rows; i++ {
		buttonBg := image.NewRGBA(image.Rect(0, 0, pixels, pixels))

		if background != nil {
			startx := (i % cols) * (pixels + padding)
			starty := (i / cols) * (pixels + padding)
			draw.Draw(buttonBg, buttonBg.Bounds(), background, image.Point{startx, starty}, draw.Src)
		}

		var w Widget
		if k, found := keyMap[uint8(i)]; found {
			w = NewWidget(k.Index, k.Widget.ID, k.Action, k.ActionHold, buttonBg, k.Widget.Config)
		} else {
			w = NewWidget(uint8(i), "empty", nil, nil, buttonBg, nil)
		}
		d.Widgets = append(d.Widgets, w)
	}

	return &d, nil
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
		log.Println("Keyboard emulation is disabled!")
		return
	}

	kk := strings.Split(keys, "-")
	for i, k := range kk {
		kc, err := strconv.Atoi(strings.TrimSpace(k))
		if err != nil {
			log.Fatalf("%s is not a valid keycode: %s", k, err)
		}

		if i+1 < len(kk) {
			keyboard.KeyDown(kc)
			defer keyboard.KeyUp(kc)
		} else {
			keyboard.KeyPress(kc)
		}
	}
}

// emulates a clipboard paste.
func emulateClipboard(text string) {
	err := clipboard.WriteAll(text)
	if err != nil {
		log.Fatalf("Pasting to clipboard failed: %s", err)
	}

	// paste the string
	emulateKeyPress("29-47") // ctrl-v
}

// executes a dbus method.
func executeDBusMethod(object, path, method, args string) {
	call := dbusConn.Object(object, dbus.ObjectPath(path)).Call(method, 0, args)
	if call.Err != nil {
		log.Printf("dbus call failed: %s", call.Err)
	}
}

// executes a command.
func executeCommand(cmd string) {
	args := strings.Split(cmd, " ")
	c := exec.Command(args[0], args[1:]...)
	if err := c.Start(); err != nil {
		log.Printf("command failed: %s", err)
		return
	}

	if err := c.Wait(); err != nil {
		log.Printf("command failed: %s", err)
	}
}

// triggerAction triggers an action.
func (d *Deck) triggerAction(index uint8, hold bool) {
	for _, w := range d.Widgets {
		if w.Key() == index {
			var a *ActionConfig
			if hold {
				a = w.ActionHold()
			} else {
				a = w.Action()
			}

			if a != nil {
				// log.Println("Executing overloaded action")
				if a.Deck != "" {
					d, err := LoadDeck(filepath.Dir(d.File), a.Deck, &dev)
					if err != nil {
						log.Fatal(err)
					}
					err = dev.Clear()
					if err != nil {
						log.Fatal(err)
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
				if a.DBus.Method != "" {
					executeDBusMethod(a.DBus.Object, a.DBus.Path, a.DBus.Method, a.DBus.Value)
				}
				if a.Exec != "" {
					go executeCommand(a.Exec)
				}
			} else {
				w.TriggerAction()
			}
		}
	}
}

// updateWidgets updates/repaints all the widgets.
func (d *Deck) updateWidgets() {
	for _, w := range d.Widgets {
		if err := w.UpdateImage(&dev); err != nil {
			log.Fatalf("error: %v", err)
		}
		if err := w.Update(&dev); err != nil {
			log.Fatalf("error: %v", err)
		}
	}
}
