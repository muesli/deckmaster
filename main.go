package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/bendahl/uinput"
	"github.com/godbus/dbus"
	"github.com/mitchellh/go-homedir"
	"github.com/muesli/streamdeck"
)

var (
	deck *Deck

	dbusConn *dbus.Conn
	keyboard uinput.Keyboard
	shutdown = make(chan error)

	xorg          *Xorg
	recentWindows []Window

	deckFile   = flag.String("deck", "main.deck", "path to deck config file")
	device     = flag.String("device", "", "which device to use (serial number)")
	brightness = flag.Uint("brightness", 80, "brightness in percent")
	sleep      = flag.String("sleep", "", "sleep timeout")
)

const (
	longPressDuration = 350 * time.Millisecond
)

func fatal(v ...interface{}) {
	go func() { shutdown <- errors.New(fmt.Sprint(v...)) }()
}

func fatalf(format string, a ...interface{}) {
	go func() { shutdown <- fmt.Errorf(format, a...) }()
}

func expandPath(base, path string) (string, error) {
	var err error
	path, err = homedir.Expand(path)
	if err != nil {
		return "", err
	}
	if base == "" {
		return path, nil
	}

	if !filepath.IsAbs(path) {
		path = filepath.Join(base, path)
	}

	return filepath.Abs(path)
}

func eventLoop(dev *streamdeck.Device, tch chan interface{}) error {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	var keyStates sync.Map
	keyTimestamps := make(map[uint8]time.Time)

	kch, err := dev.ReadKeys()
	if err != nil {
		return err
	}
	for {
		select {
		case <-time.After(100 * time.Millisecond):
			deck.updateWidgets()

		case k, ok := <-kch:
			if !ok {
				if err = dev.Open(); err != nil {
					return err
				}
				continue
			}

			var state bool
			if ks, ok := keyStates.Load(k.Index); ok {
				state = ks.(bool)
			}
			keyStates.Store(k.Index, k.Pressed)

			if state && !k.Pressed {
				// key was released
				if time.Since(keyTimestamps[k.Index]) < longPressDuration {
					// fmt.Println("Triggering short action")
					deck.triggerAction(dev, k.Index, false)
				}
			}
			if !state && k.Pressed {
				// key was pressed
				go func() {
					// launch timer to observe keystate
					time.Sleep(longPressDuration)

					if state, ok := keyStates.Load(k.Index); ok && state.(bool) {
						// key still pressed
						// fmt.Println("Triggering long action")
						deck.triggerAction(dev, k.Index, true)
					}
				}()
			}
			keyTimestamps[k.Index] = time.Now()

		case e := <-tch:
			switch event := e.(type) {
			case WindowClosedEvent:
				handleWindowClosed(event)

			case ActiveWindowChangedEvent:
				handleActiveWindowChanged(dev, event)
			}

		case err := <-shutdown:
			return err

		case <-sigs:
			fmt.Println("Shutting down...")
			return nil
		}
	}
}

func closeDevice(dev *streamdeck.Device) {
	if err := dev.Reset(); err != nil {
		fmt.Fprintln(os.Stderr, "Unable to reset Stream Deck")
	}
	if err := dev.Close(); err != nil {
		fmt.Fprintln(os.Stderr, "Unable to close Stream Deck")
	}
}

func initDevice() (*streamdeck.Device, error) {
	d, err := streamdeck.Devices()
	if err != nil {
		return nil, err
	}
	if len(d) == 0 {
		return nil, fmt.Errorf("no Stream Deck devices found")
	}

	dev := d[0]
	if len(*device) > 0 {
		found := false
		for _, v := range d {
			if v.Serial == *device {
				dev = v
				found = true
				break
			}
		}
		if !found {
			fmt.Println("Can't find device. Available devices:")
			for _, v := range d {
				fmt.Printf("Serial %s (%d buttons)\n", v.Serial, dev.Keys)
			}
			os.Exit(1)
		}
	}

	if err := dev.Open(); err != nil {
		return nil, err
	}
	ver, err := dev.FirmwareVersion()
	if err != nil {
		return &dev, err
	}
	fmt.Printf("Found device with serial %s (%d buttons, firmware %s)\n",
		dev.Serial, dev.Keys, ver)

	if err := dev.Reset(); err != nil {
		return &dev, err
	}

	if *brightness > 100 {
		*brightness = 100
	}
	if err = dev.SetBrightness(uint8(*brightness)); err != nil {
		return &dev, err
	}

	if len(*sleep) > 0 {
		timeout, err := time.ParseDuration(*sleep)
		if err != nil {
			return &dev, err
		}

		dev.SetSleepTimeout(timeout)
	}

	return &dev, nil
}

func run() error {
	// initialize device
	dev, err := initDevice()
	if dev != nil {
		defer closeDevice(dev)
	}
	if err != nil {
		return fmt.Errorf("Unable to initialize Stream Deck: %s", err)
	}

	// initialize dbus connection
	dbusConn, err = dbus.SessionBus()
	if err != nil {
		return fmt.Errorf("Unable to connect to dbus: %s", err)
	}

	// initialize xorg connection and track window focus
	tch := make(chan interface{})
	xorg, err = Connect(os.Getenv("DISPLAY"))
	if err == nil {
		defer xorg.Close()
		xorg.TrackWindows(tch, time.Second)
	} else {
		fmt.Fprintf(os.Stderr, "Could not connect to X server: %s\n", err)
		fmt.Fprintln(os.Stderr, "Tracking window manager will be disabled!")
	}

	// initialize virtual keyboard
	keyboard, err = uinput.CreateKeyboard("/dev/uinput", []byte("Deckmaster"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not create virtual input device (/dev/uinput): %s\n", err)
		fmt.Fprintln(os.Stderr, "Emulating keyboard events will be disabled!")
	} else {
		defer keyboard.Close() //nolint:errcheck
	}

	// load deck
	deck, err = LoadDeck(dev, ".", *deckFile)
	if err != nil {
		return fmt.Errorf("Can't load deck: %s", err)
	}
	deck.updateWidgets()

	return eventLoop(dev, tch)
}

func main() {
	flag.Parse()

	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
