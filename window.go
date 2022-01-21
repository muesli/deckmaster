package main

import (
	"fmt"
)

func handleActiveWindowChanged(dev *DeviceWrapper, event ActiveWindowChangedEvent) {
	fmt.Printf("Active window changed to %s (%d, %s)\n",
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

	recentWindows = append([]Window{event.Window}, recentWindows...)
	if keys := int(dev.Keys); len(recentWindows) > keys {
		recentWindows = recentWindows[0:keys]
	}
	deck.updateWidgets(dev)
}

func handleWindowClosed(dev *DeviceWrapper, event WindowClosedEvent) {
	i := 0
	for _, rw := range recentWindows {
		if rw.ID == event.Window.ID {
			continue
		}

		recentWindows[i] = rw
		i++
	}
	recentWindows = recentWindows[:i]
	deck.updateWidgets(dev)
}
