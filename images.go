package main

import (
	"path/filepath"
)

func findImage(base, icon string) string {
	if !filepath.IsAbs(icon) {
		icon = filepath.Join(base, icon)
	}
	abs, err := filepath.Abs(icon)
	if err != nil {
		return icon
	}

	return abs
}
