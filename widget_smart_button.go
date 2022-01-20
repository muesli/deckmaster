package main

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"
)

var (
	commandSubstitutionRE = regexp.MustCompile(`\${command\[[a-zA-Z0-9_]+\]}`)
)

const (
	commandSubstitutionExpandSentinel = "!@!REGEXP_EXPAND!@!"
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
	IsNecessary(label string) bool
	IsChanged() bool
	ReplaceValue(label string) string
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
func (d *SmartButtonDependencyBase) IsNecessary(template string) bool {
	for _, r := range d.toBeReplaced {
		if strings.Contains(template, r) {
			return true
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

		brightness: math.MaxUint,
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

// SmartButtonCommandDependency is a dependency based on running a command.
type SmartButtonCommandDependency struct {
	command  string
	re       *regexp.Regexp
	interval int64
	lastRun  time.Time
	value    string
}

// NewSmartButtonCommandDependency returns a new SmartButtonCommandDependency.
func NewSmartButtonCommandDependency(
	command string,
	re *regexp.Regexp,
	interval int64,
) *SmartButtonCommandDependency {
	return &SmartButtonCommandDependency{
		command:  command,
		re:       re,
		interval: interval,
	}
}

// IsNecessary returns whether the dependency is necessary for the template.
func (d *SmartButtonCommandDependency) IsNecessary(template string) bool {
	return commandSubstitutionRE.MatchString(template)
}

// IsChanged returns true if the dependency value has changed.
func (d *SmartButtonCommandDependency) IsChanged() bool {
	if d.lastRun.IsZero() || time.Since(d.lastRun).Milliseconds() >= d.interval {
		str, err := runCommand(d.command)
		if err == nil {
			d.lastRun = time.Now()
			if d.value != str {
				d.value = str
				return true
			}
		}
	}

	return false
}

// ReplaceValue replaces the value of the dependency into the template.
func (d *SmartButtonCommandDependency) ReplaceValue(template string) string {
	template = commandSubstitutionRE.ReplaceAllStringFunc(
		template,
		func(m string) string {
			return commandSubstitutionExpandSentinel + "{" + m[10:len(m)-2] + "}"
		},
	)
	template = strings.ReplaceAll(
		strings.ReplaceAll(template, "$", "$$"),
		commandSubstitutionExpandSentinel,
		"$",
	)
	result := []byte{}
	result = d.re.ExpandString(
		result,
		template,
		d.value,
		d.re.FindStringSubmatchIndex(d.value),
	)
	return string(result)
}

// NewSmartButtonWidget returns a new SmartButtonWidget.
func NewSmartButtonWidget(bw *BaseWidget, opts WidgetConfig) (*SmartButtonWidget, error) {
	var icon, label, command, commandRegexp string
	_ = ConfigValue(opts.Config["icon"], &icon)
	_ = ConfigValue(opts.Config["label"], &label)
	_ = ConfigValue(opts.Config["command"], &command)
	_ = ConfigValue(opts.Config["commandRegexp"], &commandRegexp)
	var commandInterval int64
	_ = ConfigValue(opts.Config["commandInterval"], &commandInterval)

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

	if command != "" {
		if commandInterval <= 0 {
			commandInterval = 2000
		}
		if commandRegexp == "" {
			commandRegexp = `(.*)`
		}
		if re, err := regexp.Compile(commandRegexp); err == nil {
			w.appendDependencyIfNecessary(NewSmartButtonCommandDependency(
				command,
				re,
				commandInterval,
			))
		} else {
			fmt.Printf("Regexp /%s/ error %v\n", commandRegexp, err)
		}
	}

	return &w, nil
}

// appendDependency appends the dependency if the label requires it.
func (w *SmartButtonWidget) appendDependencyIfNecessary(d SmartButtonDependency) {
	if d.IsNecessary(w.labelTemplate) || d.IsNecessary(w.iconTemplate) {
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
