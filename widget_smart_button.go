package main

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	telemetrySubstitutionRE = regexp.MustCompile(`\${telemetry\[([^\]]+)\]}`)
)

// SmartButtonWidget is a button widget that can change dynamically.
type SmartButtonWidget struct {
	*ButtonWidget

	iconTemplate  string
	labelTemplate string
	currentIcon   string
	dependencies  []SmartButtonDependency
}

// SmartButtonDependency is some dependency of the smart button.
type SmartButtonDependency interface {
	IsNecessary(templates ...string) bool
	IsChanged() bool
	ReplaceValue(template string) string
}

// SmartButtonDependencyBase is the base structure of a dependency.
type SmartButtonDependencyBase struct {
	toBeReplaced []string
}

// NewSmartButtonDependencyBase returns a new SmartButtonDependencyBase.
func NewSmartButtonDependencyBase(toBeReplaced ...string) *SmartButtonDependencyBase {
	return &SmartButtonDependencyBase{
		toBeReplaced: toBeReplaced,
	}
}

// IsNecessary returns whether the dependency is necessary for the template.
func (d *SmartButtonDependencyBase) IsNecessary(templates ...string) bool {
	for _, t := range templates {
		for _, r := range d.toBeReplaced {
			if strings.Contains(t, r) {
				return true
			}
		}
	}
	return false
}

// IsChanged returns true if the dependency value has changed.
func (d *SmartButtonDependencyBase) IsChanged() bool {
	return false
}

// ReplaceValue replaces the value of the dependency into the template.
func (d *SmartButtonDependencyBase) ReplaceValue(template string) string {
	return template
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

		brightness: 9999, // out of range value to force an initial update
	}
}

// IsChanged returns true if the brightness has changed.
func (d *SmartButtonBrightnessDependency) IsChanged() bool {
	return d.brightness != *brightness
}

// ReplaceValue replaces the value of the dependency into the template.
func (d *SmartButtonBrightnessDependency) ReplaceValue(template string) string {
	d.brightness = *brightness
	value := fmt.Sprintf("%d", d.brightness)
	return strings.ReplaceAll(template, d.toBeReplaced[0], value)
}

// SmartButtonTelemetryDependency is a dependency based on running a command.
type SmartButtonTelemetryDependency struct {
	values map[string]string
}

// NewSmartButtonTelemetryDependency returns a new SmartButtonTelemetryDependency.
func NewSmartButtonTelemetryDependency() *SmartButtonTelemetryDependency {
	return &SmartButtonTelemetryDependency{
		values: make(map[string]string),
	}
}

// IsNecessary returns whether the dependency is necessary for the template.
func (d *SmartButtonTelemetryDependency) IsNecessary(templates ...string) bool {
	for _, t := range templates {
		for _, v := range telemetrySubstitutionRE.FindAllStringSubmatch(t, -1) {
			d.values[v[1]] = ""
		}
	}
	return len(d.values) != 0
}

// IsChanged returns true if the dependency value has changed.
func (d *SmartButtonTelemetryDependency) IsChanged() bool {
	changed := false

	for key, value := range d.values {
		newValue := telemetry[key]
		if value != newValue {
			d.values[key] = newValue
			changed = true
		}
	}

	return changed
}

// ReplaceValue replaces the value of the dependency into the template.
func (d *SmartButtonTelemetryDependency) ReplaceValue(template string) string {
	return telemetrySubstitutionRE.ReplaceAllStringFunc(
		template,
		func(m string) string {
			return d.values[m[12:len(m)-2]]
		},
	)
}

// NewSmartButtonWidget returns a new SmartButtonWidget.
func NewSmartButtonWidget(bw *BaseWidget, opts WidgetConfig) (*SmartButtonWidget, error) {
	var icon, label string
	_ = ConfigValue(opts.Config["icon"], &icon)
	_ = ConfigValue(opts.Config["label"], &label)

	opts.Config["icon"] = ""
	opts.Config["label"] = ""
	parent, err := NewButtonWidget(bw, opts)
	if err != nil {
		return nil, err
	}

	w := SmartButtonWidget{
		ButtonWidget:  parent,
		iconTemplate:  icon,
		labelTemplate: label,
	}
	w.label = ""
	w.appendDependencyIfNecessary(NewSmartButtonBrightnessDependency())
	w.appendDependencyIfNecessary(NewSmartButtonTelemetryDependency())

	return &w, nil
}

// appendDependency appends the dependency if the label requires it.
func (w *SmartButtonWidget) appendDependencyIfNecessary(d SmartButtonDependency) {
	if d.IsNecessary(w.labelTemplate, w.iconTemplate) {
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
	label := w.labelTemplate
	icon := w.iconTemplate
	for _, d := range w.dependencies {
		label = d.ReplaceValue(label)
		icon = d.ReplaceValue(icon)
	}

	w.label = label
	if icon != w.currentIcon {
		if err := w.LoadImage(icon); err == nil {
			w.currentIcon = icon
		}
	}

	return w.ButtonWidget.Update()
}
