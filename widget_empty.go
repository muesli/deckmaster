package main

import (
	"sync"

	"github.com/muesli/streamdeck"
)

type EmptyWidget struct {
	BaseWidget

	init sync.Once
}

func (w *EmptyWidget) UpdateImage(dev *streamdeck.Device) error {
	w.init.Do(func() {
		w.fg = nil
	})
	return nil
}
