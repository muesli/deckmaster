package main

import (
	"path/filepath"

	"github.com/mitchellh/go-homedir"
)

func findImage(base, icon string) string {
	exp, err := homedir.Expand(icon)
	if err != nil {
		return icon
	}
	if !filepath.IsAbs(exp) {
		exp = filepath.Join(base, exp)
	}
	abs, err := filepath.Abs(exp)
	if err != nil {
		return exp
	}

	return abs
}
