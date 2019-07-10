package main

import (
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/godbus/dbus"
)

type Deck struct {
	Widgets []Widget
}

func LoadDeck(deck string) (*Deck, error) {
	d := Deck{}
	dc, err := LoadConfig(deck)
	if err != nil {
		return nil, err
	}

	for _, k := range dc.Keys {
		w := NewWidget(k.Index, k.Widget.ID, k.Action, k.Widget.Config)
		d.Widgets = append(d.Widgets, w)
	}

	return &d, nil
}

// executes a dbus method
func executeDBusMethod(object, path, method, args string) {
	call := dbusConn.Object(object, dbus.ObjectPath(path)).Call(method, 0, args)
	if call.Err != nil {
		log.Printf("dbus call failed: %s", call.Err)
	}
}

// executes a command
func executeCommand(cmd string) {
	args := strings.Split(cmd, " ")
	c := exec.Command(args[0], args[1:]...)
	if err := c.Start(); err != nil {
		panic(err)
	}
}

func (d *Deck) triggerAction(index uint8) {
	for _, w := range d.Widgets {
		if w.Key() == index {
			a := w.Action()
			if a != nil {
				fmt.Println("Executing overwritten action")
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
				if a.DBus.Method != "" {
					executeDBusMethod(a.DBus.Object, a.DBus.Path, a.DBus.Method, a.DBus.Value)
				}
				if a.Exec != "" {
					executeCommand(a.Exec)
				}
			} else {
				w.TriggerAction()
			}
		}
	}
}

func (d *Deck) updateWidgets() {
	for _, w := range d.Widgets {
		w.Update(&dev)
	}
}
