package main

import (
	"image"
	"log"
	"os"
	"syscall"

	"github.com/muesli/streamdeck"
	"github.com/nfnt/resize"
)

type LauncherWidget struct {
	BaseWidget
	icon   string
	launch string
}

func (w *LauncherWidget) Update(dev *streamdeck.Device) {
	f, err := os.Open(w.icon)
	if err != nil {
		log.Fatal(err)
	}
	img, _, err := image.Decode(f)
	if err != nil {
		log.Fatal(err)
	}

	err = dev.SetImage(w.key, resize.Resize(72, 72, img, resize.Lanczos3))
	if err != nil {
		log.Fatal(err)
	}
}

func (w *LauncherWidget) TriggerAction() {
	var sysproc = &syscall.SysProcAttr{Noctty: true}
	var attr = os.ProcAttr{
		Dir: ".",
		Env: os.Environ(),
		Files: []*os.File{
			os.Stdin,
			nil,
			nil,
		},
		Sys: sysproc,
	}
	proc, err := os.StartProcess(w.launch, []string{w.launch}, &attr)
	if err != nil {
		log.Fatal(err)
	}
	err = proc.Release()
	if err != nil {
		log.Fatal(err)
	}
}
