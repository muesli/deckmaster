package main

import (
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/godbus/dbus"
)

// Deck is a set of widgets.
type Deck struct {
	Widgets []Widget
}

// LoadDeck loads a deck configuration.
func LoadDeck(deck string) (*Deck, error) {
	d := Deck{}
	dc, err := LoadConfig(deck)
	if err != nil {
		return nil, err
	}

	for _, k := range dc.Keys {
		w := NewWidget(k.Index, k.Widget.ID, k.Action, k.ActionHold, k.Widget.Config)
		d.Widgets = append(d.Widgets, w)
	}

	return &d, nil
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

func createArrayFixedLength(amount int, value int) []int {
	s := make([]int, amount)
	for i := range s {
		s[i] = value
	}
	return s
}

func padArray(targetLen int, padValue int, existingSlice []int) []int {
	currentLen := len(existingSlice)
	var newSlice []int
	newSlice = append(newSlice, existingSlice...)
	newEntries := createArrayFixedLength(targetLen-currentLen, padValue)

	newSlice = append(newSlice, newEntries...)

	return newSlice
}

// emulates a range of key presses
func emulateKeyPresses(keys string, delaysMs ...int) {
	if keyboard == nil {
		log.Println("Keyboard emulation is disabled!")
		return
	}

	kkp := strings.Split(keys, "/")

	usedDelays := createDelays(kkp, delaysMs)

	for i, kp := range kkp {
		println("Single Keybinding: ", kp)
		emulateKeyPress(kp)
		if i+1 < len(kkp) {
			println("Sleeping")
			time.Sleep(time.Duration(usedDelays[i]) * time.Millisecond)
		}
	}
}

func createDelays(kkp []string, delaysMs []int) []int {
	numberOfKeybindings := len(kkp)

	requiredDelays := numberOfKeybindings - 1
	var usedDelays []int
	givenDelays := len(delaysMs)
	if givenDelays == 0 {
		log.Println("Using single default delay value of ", 100, " ms")
		usedDelays = createArrayFixedLength(requiredDelays, 100)
	} else if givenDelays == 1 {
		usedDelays = createArrayFixedLength(requiredDelays, delaysMs[0])
	} else if givenDelays < requiredDelays {
		usedDelays = padArray(requiredDelays, delaysMs[givenDelays-1], delaysMs)
	} else if givenDelays == requiredDelays {
		usedDelays = delaysMs
	} else {
		log.Println("Too many delays giving. Skipping surplus ones.")
		usedDelays = delaysMs[:requiredDelays]
	}
	return usedDelays
}

func emulateKeyPressesDefaultDelay(keys string) {
	emulateKeyPresses(keys, 100)
}

// emulates a clipboard paste.
func emulateClipboard(text string) {
	err := clipboard.WriteAll(text)
	if err != nil {
		log.Fatalf("Pasting to clipboard failed: %s", err)
	}

	// paste the string
	emulateKeyPressesDefaultDelay("29-47") // ctrl-v
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
				fmt.Println("Executing overloaded action")
				if a.Deck != "" {
					d, err := LoadDeck(a.Deck)
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
					emulateKeyPresses(a.Keycode, a.DelaysMs...)
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
		w.Update(&dev)
	}
}
