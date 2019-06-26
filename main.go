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

func main() {
	x := Connect(os.Getenv("DISPLAY"))
	defer x.Close()

	tracker := make(chan interface{})
	x.TrackWindows(tracker, time.Second)

	d, err := streamdeck.Devices()
	if err != nil {
		panic(err)
	}
	for _, dev := range d {
		err := dev.Open()
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
			case k := <-kch:
				spew.Dump(k)

				if k.Pressed && int(k.Index) < len(recentWindows) {
					x.RequestActivation(recentWindows[k.Index])
				}
			case w := <-tracker:
				switch et := w.(type) {
				case WindowClosedEvent:
					idx := 0
					for _, rw := range recentWindows {
						if rw.ID == et.Window.ID {
							continue
						}

						recentWindows[idx] = rw
						idx++
					}
					recentWindows = recentWindows[:idx]
					updateRecentApps(dev)

				case ActiveWindowChangedEvent:
					fmt.Println(fmt.Sprintf("Active window changed to %s (%d, %s)",
						et.Window.Class, et.Window.ID, et.Window.Name))

					// remove dupes
					idx := 0
					for _, rw := range recentWindows {
						if rw.ID == et.Window.ID {
							continue
						}

						recentWindows[idx] = rw
						idx++
					}
					recentWindows = recentWindows[:idx]

					recentWindows = append([]Window{et.Window}, recentWindows...)
					if len(recentWindows) > 15 {
						recentWindows = recentWindows[0:15]
					}
					updateRecentApps(dev)
				}
			}
		}
	}
}
