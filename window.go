package main

import (
	"github.com/muesli/streamdeck"
)

func handleActiveWindowChanged(dev *streamdeck.Device, event ActiveWindowChangedEvent) {
	verbosef("Active window changed to %s (%d, %s)",
		event.Window.Class, event.Window.ID, event.Window.Name)

	// remove dupes
	i := 0
	for _, rw := range recentWindows {
		if rw.ID == event.Window.ID {
			continue
		}

		recentWindows[i] = rw
		i++
	}
	recentWindows = recentWindows[:i]

	keys := int(dev.Keys)
	recentWindows = append([]Window{event.Window}, recentWindows...)
	if len(recentWindows) > keys {
		recentWindows = recentWindows[0:keys]
	}
	deck.updateWidgets()
}

func handleWindowClosed(event WindowClosedEvent) {
	i := 0
	for _, rw := range recentWindows {
		if rw.ID == event.Window.ID {
			continue
		}

		recentWindows[i] = rw
		i++
	}
	recentWindows = recentWindows[:i]
	deck.updateWidgets()
}
