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
	// Version contains the application version number. It's set via ldflags
	// when building.
	Version = ""

	// CommitSHA contains the SHA of the commit that this application was built
	// against. It's set via ldflags when building.
	CommitSHA = ""

	deck *Deck

	dbusConn *dbus.Conn
	keyboard uinput.Keyboard
	shutdown = make(chan error)

	xorg          *Xorg
	recentWindows []Window

	mediaPlayers *MediaPlayers

	imageDownloader *ImageDownloader

	deckFile   = flag.String("deck", "main.deck", "path to deck config file")
	device     = flag.String("device", "", "which device to use (serial number)")
	brightness = flag.Uint("brightness", 80, "brightness in percent")
	sleep      = flag.String("sleep", "", "sleep timeout")
	verbose    = flag.Bool("verbose", false, "verbose output")
	version    = flag.Bool("version", false, "display version")
)

const (
	fadeDuration      = 250 * time.Millisecond
	longPressDuration = 350 * time.Millisecond
)

func fatal(v ...interface{}) {
	go func() { shutdown <- errors.New(fmt.Sprint(v...)) }()
}

func fatalf(format string, a ...interface{}) {
	go func() { shutdown <- fmt.Errorf(format, a...) }()
}

func verbosef(format string, a ...interface{}) {
	if !*verbose {
		return
	}

	fmt.Printf(format+"\n", a...)
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

func eventLoop(dev *streamdeck.Device, tch chan interface{}, mch chan interface{}) error {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	hup := make(chan os.Signal, 1)
	signal.Notify(hup, syscall.SIGHUP)

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
					verbosef("Triggering short action for key %d", k.Index)
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
						verbosef("Triggering long action for key %d", k.Index)
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

		case e := <-mch:
			switch event := e.(type) {
			case MediaPlayerStatusChanged:
				fmt.Fprintf(os.Stderr, "Media player event: %T %+v\n", event, event)
				handleMediaPlayerStatusChanged()
			case ActiveMediaPlayerChanged:
				fmt.Fprintf(os.Stderr, "Media player event: %T %+v\n", event, event)
				handleMediaPlayerActivePlayerChanged()
			default:
				fmt.Fprintf(os.Stderr, "Invalid event: %T %+v\n", event, event)
			}

		case err := <-shutdown:
			return err

		case <-hup:
			verbosef("Received SIGHUP, reloading configuration...")

			nd, err := LoadDeck(dev, ".", deck.File)
			if err != nil {
				verbosef("The new configuration is not valid, keeping the current one.")
				fmt.Fprintf(os.Stderr, "Configuration Error: %s\n", err)
				continue
			}

			deck = nd
			deck.updateWidgets()

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
			fmt.Fprintln(os.Stderr, "Can't find device. Available devices:")
			for _, v := range d {
				fmt.Fprintf(os.Stderr, "Serial %s (%d buttons)\n", v.Serial, dev.Keys)
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
	verbosef("Found device with serial %s (%d buttons, firmware %s)",
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

	dev.SetSleepFadeDuration(fadeDuration)
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

	// initialize mediaPlayer
	mch := make(chan interface{}, 20)
	mediaPlayers, err = NewMediaPlayers(mch)
	if err != nil {
		return fmt.Errorf("Error while initializing media players: %s", err)
	}
	err = mediaPlayers.Run()
	if err != nil {
		return fmt.Errorf("Error while running media players: %s", err)
	}

	// initialize image downloader
	imageDownloader = NewImageDownloader(1 * time.Hour)

	// load deck
	deck, err = LoadDeck(dev, ".", *deckFile)
	if err != nil {
		return fmt.Errorf("Can't load deck: %s", err)
	}
	deck.updateWidgets()

	return eventLoop(dev, tch, mch)
}

func main() {
	flag.Parse()

	if *version {
		if len(CommitSHA) > 7 {
			CommitSHA = CommitSHA[:7]
		}
		if Version == "" {
			Version = "(built from source)"
		}

		fmt.Printf("deckmaster %s", Version)
		if len(CommitSHA) > 0 {
			fmt.Printf(" (%s)", CommitSHA)
		}

		fmt.Println()
		os.Exit(0)
	}

	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
