package main

import (
	"fmt"
	"log"
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
