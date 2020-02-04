# deckmaster

[![GoDoc](https://godoc.org/github.com/golang/gddo?status.svg)](https://godoc.org/github.com/muesli/deckmaster)
[![Build Status](https://travis-ci.org/muesli/deckmaster.svg?branch=master)](https://travis-ci.org/muesli/deckmaster)
[![Go ReportCard](http://goreportcard.com/badge/muesli/deckmaster)](http://goreportcard.com/report/muesli/deckmaster)

An application to control your Elgato Stream Deck on Linux

## Installation

Make sure you have a working Go environment (Go 1.9 or higher is required).
See the [install instructions](http://golang.org/doc/install.html).

To install deckmaster, simply run:

    go get github.com/muesli/deckmaster

## System Setup

On Linux you need to set up some udev rules to be able to access the device as a
regular user. Edit `/etc/udev/rules.d/99-streamdeck.rules` and add these lines:

```
SUBSYSTEM=="usb", ATTRS{idVendor}=="0fd9", ATTRS{idProduct}=="0060", MODE:="666", GROUP="plugdev"
SUBSYSTEM=="usb", ATTRS{idVendor}=="0fd9", ATTRS{idProduct}=="0063", MODE:="666", GROUP="plugdev"
SUBSYSTEM=="usb", ATTRS{idVendor}=="0fd9", ATTRS{idProduct}=="006c", MODE:="666", GROUP="plugdev"
```

Make sure your user is part of the `plugdev` group and reload the rules with
`sudo udevadm control --reload-rules`. Unplug and replug the device and you
should be good to go.

## Configuration

You can find a few example configurations in the [decks](https://github.com/muesli/deckmaster/tree/master/decks)
directory. Edit them to your needs!

## Usage

Start `deckmaster`:

```bash
deckmaster -deck deck/main.deck
```

You can control the brightness, in percent:

```
deckmaster -brightness 50
```
