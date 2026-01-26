package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"sync"
	"syscall"
	"time"

	"github.com/godbus/dbus/v5"

	"github.com/bendahl/uinput"
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

	keyboard uinput.Keyboard
	shutdown = make(chan error)

	invalidChars = regexp.MustCompile("[[:^graph:]]+")

	pa            *PulseAudio
	xorg          *Xorg
	recentWindows []Window

	deckFileConfig   = flag.String("deck", "main.deck", "path to deck config file")
	deviceConfig     = flag.String("device", "", "which device to use (serial number)")
	brightnessConfig = flag.Uint("brightness", 80, "brightness in percent")
	sleepConfig      = flag.String("sleep", "", "sleep timeout")
	verboseConfig    = flag.Bool("verbose", false, "verbose output")
	versionConfig    = flag.Bool("version", false, "display version")
)

const (
	fadeDuration      = 250 * time.Millisecond
	longPressDuration = 350 * time.Millisecond
)

func errorLog(e error, format string, args ...interface{}) {
	if e != nil {
		message := fmt.Sprintf(format, args...)
		_, _ = fmt.Fprintf(os.Stderr, "ERROR: %s\n\t%+v\n", message, e)
	}
}

func errorLogF(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
}

func fatal(e error) {
	go func() { shutdown <- e }()
}

func verboseLog(format string, a ...interface{}) {
	if *verboseConfig {
		fmt.Printf(format+"\n", a...)
	}
}

func strip(s string) string {
	return invalidChars.ReplaceAllString(s, "")
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

func reapChildProcesses() {
	sigs := make(chan os.Signal)
	signal.Notify(sigs, syscall.SIGCHLD)

	for range sigs {
		for {
			var status syscall.WaitStatus
			pid, err := syscall.Wait4(-1, &status, syscall.WNOHANG, nil)
			if pid <= 0 || err != nil {
				break
			}
			verboseLog("reaped pid=%d status=%d", pid, status.ExitStatus())
		}
	}
}

func eventLoop(dev *streamdeck.Device, tch chan interface{}) error {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	hup := make(chan os.Signal, 1)
	signal.Notify(hup, syscall.SIGHUP)

	var keyStates sync.Map
	keyTimestamps := make(map[uint8]time.Time)

	go pa.Start()
	go reapChildProcesses()

	wch := MonitorActiveWindowChanged()

	kch, e := dev.ReadKeys()
	if e != nil {
		return e
	}
	for {
		select {
		case <-time.After(100 * time.Millisecond):
			deck.updateWidgets()

		case k, ok := <-kch:
			if !ok {
				if e = dev.Open(); e != nil {
					return e
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
					verboseLog("Triggering short action for key %d", k.Index)
					deck.triggerAction(dev, k.Index, false)
				}
			}
			if !state && k.Pressed {
				// key was pressed
				go func() {
					// launch timer to observe KeyState
					time.Sleep(longPressDuration)

					if state, ok := keyStates.Load(k.Index); ok && state.(bool) {
						// key still pressed
						verboseLog("Triggering long action for key %d", k.Index)
						deck.triggerAction(dev, k.Index, true)
					}
				}()
			}
			keyTimestamps[k.Index] = time.Now()

		case changeType := <-pa.Updates():
			playback := changeType == SinkMuteChanged || changeType == SinkChanged
			for widget := range deck.Widgets {
				w, success := widget.(MuteChangedMonitor)
				if success {
					w.MuteChanged(playback)
				}
				if changeType == SinkChanged || changeType == SourceChanged {
					w, success := widget.(AudioChangedMonitor)
					if success {
						w.AudioStreamChanged(changeType)
					}
				}
			}

		case activeWindow := <-wch:
			deck.WindowChanged(activeWindow)

		case event := <-tch:
			switch event := event.(type) {
			case WindowClosedEvent:
				handleWindowClosed(event)

			case ActiveWindowChangedEvent:
				handleActiveWindowChanged(dev, event)
			}

		case err := <-shutdown:
			return err

		case <-hup:
			verboseLog("Received SIGHUP, reloading configuration...")

			nd, e := LoadDeck(dev, ".", deck.file)
			if e != nil {
				errorLog(e, "invalid configuration")
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
	errorLog(dev.Reset(), "failed to reset Stream Deck")
	errorLog(dev.Clear(), "failed to clear the Stream Deck")
	errorLog(dev.Sleep(), "failed to sleep the Stream Deck")
	errorLog(dev.Close(), "failed to close Stream Deck")
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
	if len(*deviceConfig) > 0 {
		found := false
		for _, v := range d {
			if v.Serial == *deviceConfig {
				dev = v
				found = true
				break
			}
		}
		if !found {
			errorLogF("Can't find device. Available devices:")
			for _, v := range d {
				errorLogF("Serial %s (%d buttons)", v.Serial, dev.Keys)
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
	verboseLog("Found device with serial %s (%d buttons, firmware %s)",
		dev.Serial, dev.Keys, strip(ver))

	if err := dev.Reset(); err != nil {
		return &dev, err
	}

	if *brightnessConfig > 100 {
		*brightnessConfig = 100
	}
	if err = dev.SetBrightness(uint8(*brightnessConfig)); err != nil {
		return &dev, err
	}

	dev.SetSleepFadeDuration(fadeDuration)
	if len(*sleepConfig) > 0 {
		timeout, err := time.ParseDuration(*sleepConfig)
		if err != nil {
			return &dev, err
		}

		dev.SetSleepTimeout(timeout)
	}

	return &dev, nil
}

func run() error {
	// initialize device
	dev, e := initDevice()
	if dev != nil {
		defer closeDevice(dev)
	}
	if e != nil {
		return fmt.Errorf("failed to initialize Stream Deck: %w", e)
	}

	// initialize dbus connection
	sessionBus, e := dbus.ConnectSessionBus()
	if e != nil {
		return fmt.Errorf("failed to connect to DBus %w", e)
	}
	defer sessionBus.Close() //nolint:errcheck

	// initialize xorg connection and track window focus
	tch := make(chan interface{})
	xorg, e = Connect()
	if e == nil {
		defer xorg.Close()
		xorg.TrackWindows(tch, time.Second)
	} else {
		errorLog(e, "failed to connect to X server (Wayland?)")
	}

	// initialize virtual keyboard
	keyboard, e = uinput.CreateKeyboard("/dev/uinput", []byte("deckmaster"))
	if e != nil {
		errorLog(e, "failed to create virtual input device (/dev/uinput)")
		errorLogF("Emulating keyboard events will be disabled!")
	} else {
		defer keyboard.Close() //nolint:errcheck
	}

	// initialize PulseAudio
	pa, e = NewPulseAudio()
	if e != nil {
		errorLog(e, "failed to create PulseAudio device")
	} else {
		defer pa.Close()
	}

	// load deck
	deck, e = LoadDeck(dev, ".", *deckFileConfig)
	if e != nil {
		return fmt.Errorf("failed to load deck: %s", e)
	}
	deck.updateWidgets()

	return eventLoop(dev, tch)
}

func main() {
	flag.Parse()

	if *versionConfig {
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

	if e := run(); e != nil {
		errorLog(e, "fatal")
		os.Exit(1)
	}
}
