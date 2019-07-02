package main

import (
	"io/ioutil"
	"log"
	"strconv"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"github.com/muesli/streamdeck"
)

var (
	ttfFont *truetype.Font
)

type Widget interface {
	Key() uint8
	Update(dev *streamdeck.Device)
	Action() *ActionConfig
	TriggerAction()
}

type BaseWidget struct {
	key    uint8
	action *ActionConfig
}

func (w *BaseWidget) Key() uint8 {
	return w.key
}

func (w *BaseWidget) Action() *ActionConfig {
	return w.action
}

func NewWidget(index uint8, id string, action *ActionConfig, config map[string]string) Widget {
	bw := BaseWidget{index, action}

	switch id {
	case "recentWindow":
		i, err := strconv.ParseUint(config["window"], 10, 64)
		if err != nil {
			log.Fatal(err)
		}
		return &RecentWindowWidget{BaseWidget: bw, window: uint8(i)}
	case "top":
		return &TopWidget{bw}
	case "launcher":
		return &LauncherWidget{BaseWidget: bw, launch: config["exec"], icon: config["icon"]}
	}

	return nil
}

func init() {
	ttf, err := ioutil.ReadFile("/usr/share/fonts/TTF/Roboto-Medium.ttf")
	if err != nil {
		log.Fatal(err)
	}

	ttfFont, err = freetype.ParseFont(ttf)
	if err != nil {
		log.Fatal(err)
	}
}
