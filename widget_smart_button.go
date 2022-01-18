package main

import (
	"fmt"
	"math"
	"strings"
)

// SmartButtonWidget is a button widget that can change dynamically.
type SmartButtonWidget struct {
	*ButtonWidget

	unformattedLabel string
	dependencies     []SmartButtonDependency
}

// SmartButtonDependency is some dependency of the smart button.
type SmartButtonDependency interface {
	ToBeReplaced() string
	IsChanged() bool
	Value() string
}

// SmartButtonDependencyBase is the base structure of a dependency.
type SmartButtonDependencyBase struct {
	toBeReplaced string
}

// NewSmartButtonDependencyBase returns a new SmartButtonDependencyBase.
func NewSmartButtonDependencyBase(toBeReplaced string) *SmartButtonDependencyBase {
	return &SmartButtonDependencyBase{
		toBeReplaced: toBeReplaced,
	}
}

// ToBeReplaced returns the string that will be replaced by the dependency value.
func (d *SmartButtonDependencyBase) ToBeReplaced() string {
	return d.toBeReplaced
}

// IsChanged returns true if the dependency value has changed.
func (d *SmartButtonDependencyBase) IsChanged() bool {
	return false
}

// Value returns the value of the dependency.
func (d *SmartButtonDependencyBase) Value() string {
	return ""
}

// SmartButtonBrightnessDependency is a dependency based on the brightness setting.
type SmartButtonBrightnessDependency struct {
	*SmartButtonDependencyBase

	brightness uint
}

// NewSmartButtonBrightnessDependency returns a new SmartButtonBrightnessDependency.
func NewSmartButtonBrightnessDependency() *SmartButtonBrightnessDependency {
	return &SmartButtonBrightnessDependency{
		SmartButtonDependencyBase: NewSmartButtonDependencyBase("${brightness}"),
		brightness:                math.MaxUint,
	}
}

// IsChanged returns true if the brightness has changed.
func (d *SmartButtonBrightnessDependency) IsChanged() bool {
	return d.brightness != *brightness
}

// Value returns the brightness as a string.
func (d *SmartButtonBrightnessDependency) Value() string {
	d.brightness = *brightness
	return fmt.Sprintf("%d", d.brightness)
}

// NewSmartButtonWidget returns a new SmartButtonWidget.
func NewSmartButtonWidget(bw *BaseWidget, opts WidgetConfig) (*SmartButtonWidget, error) {
	var label string
	_ = ConfigValue(opts.Config["label"], &label)

	parent, err := NewButtonWidget(bw, opts)
	if err != nil {
		return nil, err
	}

	w := SmartButtonWidget{
		ButtonWidget:     parent,
		unformattedLabel: label,
	}
	w.label = ""
	w.appendDependencyIfNecessary(NewSmartButtonBrightnessDependency())

	return &w, nil
}

// appendDependency appends the dependency if the label requires it.
func (w *SmartButtonWidget) appendDependencyIfNecessary(d SmartButtonDependency) {
	if strings.Contains(w.unformattedLabel, d.ToBeReplaced()) {
		w.dependencies = append(w.dependencies, d)
	}
}

// RequiresUpdate returns true when the widget wants to be repainted.
func (w *SmartButtonWidget) RequiresUpdate() bool {
	changed := false
	for _, d := range w.dependencies {
		changed = d.IsChanged() || changed
	}
	return changed || w.ButtonWidget.RequiresUpdate()
}

// Update renders the widget.
func (w *SmartButtonWidget) Update() error {
	label := w.unformattedLabel
	for _, d := range w.dependencies {
		label = strings.Replace(label, d.ToBeReplaced(), d.Value(), -1)
	}

	w.label = label

	return w.ButtonWidget.Update()
}
