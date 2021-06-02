# deckmaster

[![Latest Release](https://img.shields.io/github/release/muesli/deckmaster.svg)](https://github.com/muesli/deckmaster/releases)
[![Build Status](https://github.com/muesli/deckmaster/workflows/build/badge.svg)](https://github.com/muesli/deckmaster/actions)
[![Go ReportCard](https://goreportcard.com/badge/muesli/deckmaster)](https://goreportcard.com/report/muesli/deckmaster)
[![GoDoc](https://godoc.org/github.com/golang/gddo?status.svg)](https://pkg.go.dev/github.com/muesli/deckmaster)

An application to control your Elgato Stream Deck on Linux

## Features

- Multiple pages & navigation between decks
- Buttons (icons & text)
- Background images
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

### Widgets

#### Button

A simple button that can display an image and/or a label.

```
[keys.widget]
  id = "button"
  [keys.widget.config]
    icon = "/some/image.png"
    label = "My Button"
```

#### Recent Window (requires X11)

Displays the icon of a recently used window/application. Pressing the button
activates the window.

```
[keys.widget]
  id = "recentWindow"
  [keys.widget.config]
    window = "1"
```

#### Time

A flexible widget that can display the current time or date.

```
[keys.widget]
  id = "time"
  [keys.widget.config]
    format = "%H;%i;%s"
    font = "bold;regular;thin"
```

Values for `format` are:

| %   | gets replaced with                                                 |
| --- | ------------------------------------------------------------------ |
| %Y  | A full numeric representation of a year, 4 digits                  |
| %y  | A two digit representation of a year                               |
| %F  | A full textual representation of a month, such as January or March |
| %M  | A short textual representation of a month, three letters           |
| %m  | Numeric representation of a month, with leading zeros              |
| %l  | A full textual representation of the day of the week               |
| %D  | A textual representation of a day, three letters                   |
| %d  | Day of the month, 2 digits with leading zeros                      |
| %h  | 12-hour format of an hour with leading zeros                       |
| %H  | 24-hour format of an hour with leading zeros                       |
| %i  | Minutes with leading zeros                                         |
| %s  | Seconds with leading zeros                                         |
| %a  | Lowercase Ante meridiem and Post meridiem                          |
| %t  | Timezone abbreviation                                              |

#### Top

This widget shows the current CPU or memory utilization as a bar graph.

```
[keys.widget]
  id = "top"
  [keys.widget.config]
    mode = "cpu"
    fillColor = "#d497de"
```

There are two values for `mode`: `cpu` and `memory`.

### Background Image

You can configure each deck to display an individual wallpaper behind its
widgets:

```
background = "/some/image.png"
```

### Actions

You can hook up any key with one of several actions:

#### Run a command

```
[keys.action]
  exec = "some_command --with-parameters"
```

#### Emulate a key-press

```
[keys.action]
  keycode = "114"
```

#### Paste to clipboard

```
[keys.action]
  paste = "a text"
```

#### Trigger a dbus call

```
[keys.action]
  [dbus]
    object = "object"
    path = "path"
    method = "method"
    value = "value"
```

## Usage

Start `deckmaster`:

```bash
deckmaster -deck deck/main.deck
```

You can control the brightness, in percent:

```
deckmaster -brightness 50
```
