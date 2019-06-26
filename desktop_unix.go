// +build linux

package main

import (
	"bytes"
	"errors"
	"image"
	"log"
	"time"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/screensaver"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/BurntSushi/xgbutil"
	"github.com/BurntSushi/xgbutil/ewmh"
	"github.com/BurntSushi/xgbutil/xgraphics"
)

type Xorg struct {
	conn         *xgb.Conn
	util         *xgbutil.XUtil
	root         xproto.Window
	activeAtom   *xproto.InternAtomReply
	netNameAtom  *xproto.InternAtomReply
	nameAtom     *xproto.InternAtomReply
	classAtom    *xproto.InternAtomReply
	activeWindow Window
}

type ActiveWindowChangedEvent struct {
	Window Window
}

type WindowClosedEvent struct {
	Window Window
}

type Window struct {
	ID    uint32
	Class string
	Name  string
	Icon  image.Image
}

var (
	ErrNoValue = errors.New("empty value")
	ErrNoClass = errors.New("empty class")
)

func Connect(display string) Xorg {
	var x Xorg
	var err error

	x.conn, err = xgb.NewConnDisplay(display)
	if err != nil {
		log.Fatal("xgb:", err)
	}

	x.util, err = xgbutil.NewConnDisplay(display)
	if err != nil {
		log.Fatal(err)
	}

	err = screensaver.Init(x.conn)
	if err != nil {
		log.Fatal("screensaver:", err)
	}

	setup := xproto.Setup(x.conn)
	x.root = setup.DefaultScreen(x.conn).Root

	drw := xproto.Drawable(x.root)
	screensaver.SelectInput(x.conn, drw, screensaver.EventNotifyMask)

	x.activeAtom = x.atom("_NET_ACTIVE_WINDOW")
	x.netNameAtom = x.atom("_NET_WM_NAME")
	x.nameAtom = x.atom("WM_NAME")
	x.classAtom = x.atom("WM_CLASS")

	x.spy(x.root)
	return x
}

func (x Xorg) Close() {
	x.util.Conn().Close()
	x.conn.Close()
}

func (x *Xorg) TrackWindows(ch chan interface{}, timeout time.Duration) {
	if win, ok := x.window(); ok {
		x.activeWindow = win

		if ch != nil {
			go func() {
				ch <- ActiveWindowChangedEvent{
					Window: win,
				}
			}()
		}
	}

	events := make(chan xgb.Event, 1)
	go x.waitForEvent(events)

	go func() {
		for {
			select {
			case event := <-events:
				switch e := event.(type) {
				case xproto.DestroyNotifyEvent:
					ch <- WindowClosedEvent{
						Window: Window{
							ID: uint32(e.Window),
						},
					}

				case xproto.PropertyNotifyEvent:
					if win, ok := x.window(); ok {
						if win.ID != x.activeWindow.ID {
							x.activeWindow = win
							if ch != nil {
								go func() {
									ch <- ActiveWindowChangedEvent{
										Window: win,
									}
								}()
							}
						}
						// Wakeup
					}
				case screensaver.NotifyEvent:
					switch e.State {
					case screensaver.StateOn:
						// Snooze(x.queryIdle())
					default:
						// Wakeup
					}
				}
			case <-time.After(timeout):
				// Snooze(x.queryIdle())
			}
		}
	}()
}

// ActiveWindow returns the currently active window
func (x Xorg) ActiveWindow() Window {
	return x.activeWindow
}

func (x Xorg) RequestActivation(w Window) error {
	return ewmh.ActiveWindowReq(x.util, xproto.Window(w.ID))
}

func (x Xorg) atom(aname string) *xproto.InternAtomReply {
	a, err := xproto.InternAtom(x.conn, true, uint16(len(aname)), aname).Reply()
	if err != nil {
		log.Fatal("atom:", err)
	}
	return a
}

func (x Xorg) property(w xproto.Window, a *xproto.InternAtomReply) (*xproto.GetPropertyReply, error) {
	return xproto.GetProperty(x.conn, false, w, a.Atom, xproto.GetPropertyTypeAny, 0, (1<<32)-1).Reply()
}

func (x Xorg) active() xproto.Window {
	p, err := x.property(x.root, x.activeAtom)
	if err != nil {
		return x.root
	}
	return xproto.Window(xgb.Get32(p.Value))
}

func (x Xorg) name(w xproto.Window) (string, error) {
	name, err := x.property(w, x.netNameAtom)
	if err != nil {
		return "", err
	}
	if string(name.Value) == "" {
		name, err = x.property(w, x.nameAtom)
		if err != nil {
			return "", err
		}
		if string(name.Value) == "" {
			return "", ErrNoValue
		}
	}
	return string(name.Value), nil
}

func (x Xorg) icon(w xproto.Window) (image.Image, error) {
	icon, err := xgraphics.FindIcon(x.util, w, 64, 64)
	if err != nil {
		log.Printf("Could not find icon for window %d.", w)
		return nil, err
	}

	return icon, nil
}

func (x Xorg) class(w xproto.Window) (string, error) {
	class, err := x.property(w, x.classAtom)
	if err != nil {
		return "", err
	}
	zero := []byte{0}
	s := bytes.Split(bytes.TrimSuffix(class.Value, zero), zero)
	if l := len(s); l > 0 && len(s[l-1]) != 0 {
		return string(s[l-1]), nil
	}
	return "", ErrNoClass
}

func (x Xorg) window() (Window, bool) {
	id := x.active()
	/* skip invalid window id */
	if id == 0 {
		return Window{}, false
	}
	class, err := x.class(id)
	if err != nil {
		return Window{}, false
	}
	name, err := x.name(id)
	if err != nil {
		return Window{}, false
	}
	icon, err := x.icon(id)
	if err != nil {
		return Window{}, false
	}
	x.spy(id)

	return Window{
		ID:    uint32(id),
		Class: class,
		Name:  name,
		Icon:  icon,
	}, true
}

func (x Xorg) spy(w xproto.Window) {
	xproto.ChangeWindowAttributes(x.conn, w, xproto.CwEventMask,
		[]uint32{xproto.EventMaskPropertyChange | xproto.EventMaskStructureNotify})
}

func (x Xorg) waitForEvent(events chan<- xgb.Event) {
	for {
		ev, err := x.conn.WaitForEvent()
		if err != nil {
			log.Println("wait for event:", err)
			continue
		}
		events <- ev
	}
}

func (x Xorg) queryIdle() time.Duration {
	info, err := screensaver.QueryInfo(x.conn, xproto.Drawable(x.root)).Reply()
	if err != nil {
		log.Println("query idle:", err)
		return 0
	}
	return time.Duration(info.MsSinceUserInput) * time.Millisecond
}
