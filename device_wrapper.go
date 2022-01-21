package main

import (
	"time"

	"github.com/muesli/streamdeck"
)

// DeviceWrapper wraps a streamdeck.Device and give it additional functions.
type DeviceWrapper struct {
	*streamdeck.Device

	asleep         bool
	lastActionTime time.Time
}

// WrapDevice wraps a streamdeck.Device and returns it.
func WrapDevice(dev *streamdeck.Device) *DeviceWrapper {
	return &DeviceWrapper{
		Device: dev,
		asleep: false,
	}
}

// startTimer starts the timer.
func (dev *DeviceWrapper) startTimer() {
	dev.asleep = false
	dev.lastActionTime = time.Now()
}

// tick is to be called on each clock tick to update the device.
func (dev *DeviceWrapper) tick(deck *Deck) {
	if !dev.asleep {
		if *timeout == 0 || time.Since(dev.lastActionTime).Minutes() < float64(*timeout) {
			deck.updateWidgets(dev)
		} else {
			dev.sleep()
		}
	}
}

// isAsleep returns whether the device is currently asleep.
func (dev *DeviceWrapper) isAsleep() bool {
	return dev.asleep
}

// triggerAction wakes up the device if it was asleep and triggers the action if the device was already awake.
func (dev *DeviceWrapper) triggerAction(deck *Deck, index uint8, hold bool) {
	dev.lastActionTime = time.Now()
	if dev.asleep {
		// wake up
		dev.asleep = false
		if err := dev.SetBrightness(uint8(*brightness)); err != nil {
			fatalf("error: %v\n", err)
		}
		deck.forceUpdateWidgets(true)
		// don't perform the action!
	} else {
		deck.triggerAction(dev, index, hold)
	}
}

// sleep causes the device to go into "sleep" mode (dim and blank).
func (dev *DeviceWrapper) sleep() {
	dev.asleep = true
	if err := dev.SetBrightness(0); err != nil {
		fatalf("error: %v\n", err)
	}
	if err := dev.Clear(); err != nil {
		fatalf("error: %v\n", err)
	}
}
