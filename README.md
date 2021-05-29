# deckmaster

[![Latest Release](https://img.shields.io/github/release/muesli/deckmaster.svg)](https://github.com/muesli/deckmaster/releases)
[![Build Status](https://github.com/muesli/deckmaster/workflows/build/badge.svg)](https://github.com/muesli/deckmaster/actions)
[![Go ReportCard](https://goreportcard.com/badge/muesli/deckmaster)](https://goreportcard.com/report/muesli/deckmaster)
[![GoDoc](https://godoc.org/github.com/golang/gddo?status.svg)](https://pkg.go.dev/github.com/muesli/deckmaster)

An application to control your Elgato Stream Deck on Linux

## Features

- Multiple pages & navigation between decks
- Buttons (icons & text)
- Brightness control
- Supports different actions for short & long presses
- Comes with a collection of widgets:
    - Buttons
    - Clock
    - Date
    - CPU/Mem usage
    - Recently used windows (X11-only)
- Lets you trigger several actions:
    - Run commands
    - Emulate a key-press
    - Paste to clipboard
    - Trigger a dbus call

## Installation

### Packages

- Arch Linux: [deckmaster](https://aur.archlinux.org/packages/deckmaster/)
- [Packages](https://github.com/muesli/deckmaster/releases) in Alpine, Debian & RPM formats
- [Binaries](https://github.com/muesli/deckmaster/releases) for various architectures

### From source

Make sure you have a working Go environment (Go 1.12 or higher is required).
See the [install instructions](https://golang.org/doc/install.html).

To install deckmaster, simply run:

    git clone https://github.com/muesli/deckmaster.git
    cd deckmaster
    go build

## System Setup

On Linux you need to set up some udev rules to be able to access the device as a
regular user. Edit `/etc/udev/rules.d/99-streamdeck.rules` and add these lines:

```
SUBSYSTEM=="usb", ATTRS{idVendor}=="0fd9", ATTRS{idProduct}=="0060", MODE:="666", GROUP="plugdev", SYMLINK+="streamdeck"
SUBSYSTEM=="usb", ATTRS{idVendor}=="0fd9", ATTRS{idProduct}=="006d", MODE:="666", GROUP="plugdev", SYMLINK+="streamdeck"
SUBSYSTEM=="usb", ATTRS{idVendor}=="0fd9", ATTRS{idProduct}=="0063", MODE:="666", GROUP="plugdev", SYMLINK+="streamdeck-mini"
SUBSYSTEM=="usb", ATTRS{idVendor}=="0fd9", ATTRS{idProduct}=="006c", MODE:="666", GROUP="plugdev", SYMLINK+="streamdeck-xl"
```

Make sure your user is part of the `plugdev` group and reload the rules with
`sudo udevadm control --reload-rules`. Unplug and replug the device and you
should be good to go.

### Starting deckmaster automatically

If you want deckmaster to be started automatically upon device plugin, you can use systemd path activation, adding `streamdeck.path` and `streamdeck.service` files to `$HOME/.config/systemd/user`.

`streamdeck.path` contents:

```ini
[Unit]
Description="Stream Deck Device Path"

[Path]
# the device name will be different if you use streamdeck-mini or streamdeck-xl
PathExists=/dev/streamdeck
Unit=streamdeck.service

[Install]
WantedBy=multi-user.target
```

`streamdeck.service` contents:

```ini
[Unit]
Description=Deckmaster Service

[Service]
# adjust the path to deckmaster and .deck file to suit your needs
ExecStart=/usr/local/bin/deckmaster --deck path-to/some.deck
Restart=on-failure

[Install]
WantedBy=default.target
```

Then enable and start the `streamdeck.path` unit:

```
systemctl --user enable streamdeck.path
systemctl --user start streamdeck.path
```

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
