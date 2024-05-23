# deckmaster

[![Latest Release](https://img.shields.io/github/release/muesli/deckmaster.svg?style=for-the-badge)](https://github.com/muesli/deckmaster/releases)
[![Go Doc](https://img.shields.io/badge/godoc-reference-blue.svg?style=for-the-badge)](https://pkg.go.dev/github.com/muesli/deckmaster)
[![Software License](https://img.shields.io/badge/license-MIT-blue.svg?style=for-the-badge)](/LICENSE)
[![Build Status](https://img.shields.io/github/actions/workflow/status/muesli/deckmaster/build.yml?branch=master&style=for-the-badge)](https://github.com/muesli/deckmaster/actions)
[![Go ReportCard](https://goreportcard.com/badge/github.com/muesli/deckmaster?style=for-the-badge)](https://goreportcard.com/report/muesli/deckmaster)

An application to control your Elgato Stream Deck on Linux

## Features

- Multiple pages & navigation between decks
- Buttons (icons & text)
- Background images
- Brightness control
- Supports different actions for short & long presses
- Comes with a collection of widgets:
    - Buttons
    - Time (with formatting)
    - CPU/Mem usage
    - Weather
    - Command output
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

Make sure you have a working Go environment (Go 1.17 or higher is required).
See the [install instructions](https://golang.org/doc/install.html).

To install deckmaster, simply run:

    git clone https://github.com/muesli/deckmaster.git
    cd deckmaster
    go build

## System Setup

On Linux you need to set up some `udev` rules to be able to access the device as
a regular user. Edit `/etc/udev/rules.d/99-streamdeck.rules` and add these lines:

```
SUBSYSTEM=="usb", ATTRS{idVendor}=="0fd9", ATTRS{idProduct}=="0060", MODE:="666", GROUP="plugdev", SYMLINK+="streamdeck"
SUBSYSTEM=="usb", ATTRS{idVendor}=="0fd9", ATTRS{idProduct}=="006d", MODE:="666", GROUP="plugdev", SYMLINK+="streamdeck"
SUBSYSTEM=="usb", ATTRS{idVendor}=="0fd9", ATTRS{idProduct}=="0080", MODE:="666", GROUP="plugdev", SYMLINK+="streamdeck"
SUBSYSTEM=="usb", ATTRS{idVendor}=="0fd9", ATTRS{idProduct}=="0063", MODE:="666", GROUP="plugdev", SYMLINK+="streamdeck-mini"
SUBSYSTEM=="usb", ATTRS{idVendor}=="0fd9", ATTRS{idProduct}=="0090", MODE:="666", GROUP="plugdev", SYMLINK+="streamdeck-mini"
SUBSYSTEM=="usb", ATTRS{idVendor}=="0fd9", ATTRS{idProduct}=="006c", MODE:="666", GROUP="plugdev", SYMLINK+="streamdeck-xl"
```

Make sure your user is part of the `plugdev` group and reload the rules with
`sudo udevadm control --reload-rules`. Unplug and re-plug the device, and you
should be good to go.

### Starting deckmaster automatically

If you want deckmaster to be started automatically upon device plugin, you can
use systemd path activation, adding `streamdeck.path` and `streamdeck.service`
files to `$HOME/.config/systemd/user`.

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
ExecReload=kill -HUP $MAINPID

[Install]
WantedBy=default.target
```

Then enable and start the `streamdeck.path` unit:

```bash
systemctl --user enable streamdeck.path
systemctl --user start streamdeck.path
```

## Usage

Start `deckmaster` with the initial deck configuration you want to load:

```bash
deckmaster -deck deck/main.deck
```

You can control the brightness, in percent:

```bash
deckmaster -brightness 50
```

Control a specific streamdeck:

```bash
deckmaster -device [serial number]
```

Set a sleep timeout after which the screen gets turned off:

```bash
deckmaster -sleep 10m
```

## Configuration

You can find a few example configurations in the [decks](https://github.com/muesli/deckmaster/tree/master/decks)
directory. Edit them to your needs!

### Widgets

Any widget is build up the following way:

```toml
[[keys]]
  index = 0
```

`index` needs to be present in every widget and describes the position of the
widget on the streamdeck. `index` is 0-indexed and counted from top to bottom
and left to right.

#### Update interval for widgets

Optionally, you can configure an update `interval` for each widget:

```toml
[keys.widget]
  id = "button"
  interval = 500 # optional
```

The attribute `interval` defines the time in `ms` between two consecutive
updates of a widget.

#### Button

A simple button that can display an image and/or a label.

```toml
[keys.widget]
  id = "button"
  [keys.widget.config]
    icon = "/some/image.png" # optional
    label = "My Button" # optional
    fontsize = 10.0 # optional
    color = "#fefefe" # optional
    flatten = true # optional
```

If `flatten` is `true` all opaque pixels of the icon will have the color `color`.

#### Recent Window (requires X11)

Displays the icon of a recently used window/application. Pressing the button
activates the window.

```toml
[keys.widget]
  id = "recentWindow"
  [keys.widget.config]
    window = 1
    showTitle = true # optional
```

If `showTitle` is `true`, the title of the window will be displayed below the
window icon.

#### Time

A flexible widget that can display the current time or date.

```toml
[keys.widget]
  id = "time"
  [keys.widget.config]
    format = "%H;%i;%s"
    font = "bold;regular;thin" # optional
    color = "#fefefe" # optional
    layout = "0x0+72x24;0x24+72x24;0x48+72x24" # optional
```

With `layout` custom layouts can be definded in the format `[posX]x[posY]+[width]x[height]`.

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

```toml
[keys.widget]
  id = "top"
  [keys.widget.config]
    mode = "cpu"
    color = "#fefefe" # optional
    fillColor = "#d497de" # optional
```

There are two values for `mode`: `cpu` and `memory`.

#### Command

A widget that displays the output of commands.

```toml
[keys.widget]
  id = "command"
  [keys.widget.config]
    command = "echo 'Files:'; ls -a ~ | wc -l"
    font = "regular;bold" # optional
    color = "#fefefe" # optional
    layout = "0x0+72x20;0x20+72x52" # optional
```

#### Weather

A widget that displays the weather condition and temperature.

```toml
[keys.widget]
  id = "weather"
  [keys.widget.config]
    location = "MyCity" # optional
    unit = "celsius" # optional
    color = "#fefefe" # optional
    flatten = true # optional
    theme = "openmoji" # optional
```

The supported location types can be found [here](http://wttr.in/:help). The unit
has to be either `celsius` or `fahrenheit`. If `flatten` is `true` all opaque
pixels of the condition icon will have the color `color`. In case `theme` is set
corresponding icons with correct names need to be placed in
`~/.local/share/deckmaster/themes/[theme]`. The default icons with their
respective names can be found [here](https://github.com/muesli/deckmaster/tree/master/assets/weather).

### Actions

You can hook up any key with several actions. A regular keypress will trigger
the widget's configured `keys.action`, while holding the key will trigger
`keys.action_hold`.

#### Switch deck

```toml
[keys.action]
  deck = "relative/path/to/another.deck"
```

#### Run a command

```toml
[keys.action]
  exec = "some_command --with-parameters"
```

#### Emulate key-presses

```toml
[keys.action]
  keycode = "Leftctrl-C"
```

Emulate a series of key-presses with delay in between:

```toml
[keys.action]
  keycode = "Leftctrl-X+500 / Leftctrl-V / Num1"
```

A list of available `keycodes` can be found here: [keycodes](https://github.com/muesli/deckmaster/blob/master/keycodes.go)

#### Paste to clipboard

```toml
[keys.action]
  paste = "some text"
```

#### Trigger a dbus call

```toml
[keys.action]
  [dbus]
    object = "object"
    path = "path"
    method = "method"
    value = "value"
```

#### Device actions

Increase the brightness. If no value is specified, it will be increased by 10%:

```toml
[keys.action]
  device = "brightness+5"
```

Decrease the brightness. If no value is specified, it will be decreased by 10%:

```toml
[keys.action]
  device = "brightness-5"
```

Set the brightness to a specific value between 0 and 100:

```toml
[keys.action]
  device = "brightness=50"
```

Put the device into sleep mode, blanking the screen until the next key gets
pressed:

```toml
[keys.action]
  device = "sleep"
```

### Background Image

You can configure each deck to display an individual wallpaper behind its
widgets:

```toml
background = "/some/image.png"
```

### Re-using another deck's configuration

If you specify a `parent` inside a deck's configuration, it will inherit all
of the parent's settings that are not overwritten by the deck's own settings.
This even works recursively:

```toml
parent = "another.deck"
```

## More Decks!

* [deckmaster-emojis](https://github.com/muesli/deckmaster-emojis), an Emoji keyboard deck
* [deckmaster-helldivers2](https://github.com/boj/deckmaster-helldivers2), a deck for calling down Helldivers 2 game stratagems

Made your own useful decks? Submit a pull request!

## Feedback

Got some feedback or suggestions? Please open an issue or drop me a note!

* [Twitter](https://twitter.com/mueslix)
* [The Fediverse](https://mastodon.social/@fribbledom)
