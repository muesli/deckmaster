package main

import (
	"fmt"
	"image"
	"os"
	"time"

	"image/draw"
	_ "image/jpeg"
	_ "image/png"

	"github.com/davecgh/go-spew/spew"
	"github.com/muesli/streamdeck"
)

var (
	recentWindows []Window
)

func updateRecentApps(dev streamdeck.Device) {
	for i := 0; i < int(dev.Columns)*int(dev.Rows); i++ {
		img := image.NewRGBA(image.Rect(0, 0, 72, 72))

		if i < len(recentWindows) {
			draw.Draw(img, image.Rect(4, 4, 68, 68), recentWindows[i].Icon, image.Point{0, 0}, draw.Src)
		}

		err := dev.SetImage(uint8(i), img)
		if err != nil {
			panic(err)
		}
	}
}

func handleActiveWindowChanged(dev streamdeck.Device, event ActiveWindowChangedEvent) {
	fmt.Println(fmt.Sprintf("Active window changed to %s (%d, %s)",
		event.Window.Class, event.Window.ID, event.Window.Name))

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
	if len(recentWindows) > 15 {
		recentWindows = recentWindows[0:15]
	}
	updateRecentApps(dev)
}

func handleWindowClosed(dev streamdeck.Device, event WindowClosedEvent) {
	i := 0
	for _, rw := range recentWindows {
		if rw.ID == event.Window.ID {
			continue
		}

		recentWindows[i] = rw
		i++
	}
	recentWindows = recentWindows[:i]
	updateRecentApps(dev)

}

func main() {
	x := Connect(os.Getenv("DISPLAY"))
	defer x.Close()

	tch := make(chan interface{})
	x.TrackWindows(tch, time.Second)

	d, err := streamdeck.Devices()
	if err != nil {
		panic(err)
	}
	if len(d) == 0 {
		fmt.Println("No Stream Deck devices found.")
		return
	}
	dev := d[0]

	err = dev.Open()
	if err != nil {
		panic(err)
	}
	ver, err := dev.FirmwareVersion()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Found device with serial %s (firmware %s)\n",
		dev.Serial, ver)

	err = dev.Reset()
	if err != nil {
		panic(err)
	}
	err = dev.SetBrightness(80)
	if err != nil {
		panic(err)
	}

	kch, err := dev.ReadKeys()
	if err != nil {
		panic(err)
	}
	for {
		select {
		case k, ok := <-kch:
			if !ok {
				err = dev.Open()
				if err != nil {
					panic(err)
				}
				continue
			}
			spew.Dump(k)

			if k.Pressed && int(k.Index) < len(recentWindows) {
				x.RequestActivation(recentWindows[k.Index])
			}
		case e := <-tch:
			switch event := e.(type) {
			case WindowClosedEvent:
				handleWindowClosed(dev, event)

			case ActiveWindowChangedEvent:
				handleActiveWindowChanged(dev, event)
			}
		}
	}
}
