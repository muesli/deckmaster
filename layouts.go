package main

import (
	"fmt"
	"image"
	"os"
	"strconv"
	"strings"
)

// Layout contains the data to represent the layout of the widget.
type Layout struct {
	frames []image.Rectangle
	size   int
	margin int
	height int
}

// NewLayout returns a new Layout with the accoriding size.
func NewLayout(size int) *Layout {
	margin := size / 18
	height := size - (margin * 2)

	return &Layout{
		size:   size,
		margin: margin,
		height: height,
	}
}

// DefaultLayout returns a layout that is evenly split in frameCount horziontal containers.
func (l *Layout) DefaultLayout(frameCount int) []image.Rectangle {
	if frameCount < 1 {
		frameCount = 1
	}
	for i := 0; i < frameCount; i++ {
		frame := l.defaultFrame(frameCount, i)
		l.frames = append(l.frames, frame)
	}
	return l.frames
}

// FormatLayout returns a layout that is formatted according to frameReps.
func (l *Layout) FormatLayout(frameReps []string, frameCount int) []image.Rectangle {
	if frameCount < 1 {
		frameCount = 1
	}
	for i := 0; i < frameCount; i++ {
		if len(frameReps) < i+1 {
			frame := l.defaultFrame(frameCount, i)
			l.frames = append(l.frames, frame)
			continue
		}

		frame, err := formatFrame(frameReps[i])
		if err != nil {
			fmt.Fprintln(os.Stderr, "using default frame:", err)
			frame = l.defaultFrame(frameCount, i)
		}
		l.frames = append(l.frames, frame)
	}
	return l.frames
}

// Returns the Rectangle representing the index-th horizontal cell.
func (l *Layout) defaultFrame(cells int, index int) image.Rectangle {
	lower := l.margin + (l.height/cells)*index
	upper := l.margin + (l.height/cells)*(index+1)
	return image.Rect(0, lower, l.size, upper)
}

// Converts the string representation of a rectangle into a image.Rectangle.
func formatFrame(layout string) (image.Rectangle, error) {
	split := strings.Split(layout, "+")
	if len(split) < 2 {
		return image.Rectangle{}, fmt.Errorf("invalid rectangle format")
	}
	position, errP := formatCoord(split[0])
	if errP != nil {
		return image.Rectangle{}, errP
	}
	extent, errE := formatCoord(split[1])
	if errE != nil {
		return image.Rectangle{}, errE
	}

	return image.Rectangle{position, position.Add(extent)}, nil
}

// Converts the string representation of a point into a image.Point.
func formatCoord(coords string) (image.Point, error) {
	split := strings.Split(coords, "x")
	if len(split) < 2 {
		return image.Point{}, fmt.Errorf("invalid point format")
	}
	posX, errX := strconv.Atoi(split[0])
	posY, errY := strconv.Atoi(split[1])
	if errX != nil || errY != nil {
		return image.Point{}, fmt.Errorf("invalid point format")
	}
	return image.Pt(posX, posY), nil
}
