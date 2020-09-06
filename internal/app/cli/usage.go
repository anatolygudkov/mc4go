// Copyright (c) 2020 anatolygudkov. All rights reserved.
// Use of this source code is governed by MIT license
// that can be found in the LICENSE file.
package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"unicode"
)

const (
	screenWidth               = 80
	usageColumnsWidthFactor   = 0.6
	optionsColumnsWidthFactor = 0.5

	columnSpacing = 2
	columnSpace   = "  "
	twoWs         = "  "
	nextLine      = " \\"

	notWsState = 0
	wsState    = 1
)

// Usage presents information about how to use the application,
// including its name, command/executable name, version of the application,
// available options and examples of usage.
type Usage struct {
	name    string
	command string
	options *Options

	usages      []descriptedItem
	version     string
	description string
}

// NewUsage creates new instance of Usage with specified name and options.
func NewUsage(name string, options *Options) (u *Usage, err error) {
	exec, err := os.Executable()
	if err != nil {
		return nil, err
	}
	exec, err = filepath.EvalSymlinks(exec)
	if err != nil {
		return nil, err
	}
	u = &Usage{
		name:    name,
		command: path.Base(exec),
		options: options,
		usages:  make([]descriptedItem, 0, 10),
	}
	return u, nil
}

// AddUsage adds an example of usage with a description.
func (u *Usage) AddUsage(arguments, description string) {
	di := *newDescriptedItem(fmt.Sprintf("%s %s", u.command, arguments), description)
	u.usages = append(u.usages, di)
}

// SetVersion sets the version's information.
func (u *Usage) SetVersion(version string) {
	u.version = version
}

// SetDescription sets description of the application.
func (u *Usage) SetDescription(description string) {
	u.description = description
}

// Write writes formatted usage info into io.StringWriter.
func (u *Usage) Write(sw io.StringWriter) error {
	if _, err := sw.WriteString(u.name); err != nil {
		return err
	}

	if u.version != "" {
		if _, err := sw.WriteString(fmt.Sprintf(" - %s", u.version)); err != nil {
			return err
		}
	}

	if _, err := sw.WriteString("\n\n"); err != nil {
		return err
	}

	if u.description != "" {
		ww, err := newWordWrapper([]rune(u.description), screenWidth)
		if err != nil {
			return err
		}
		for v := ww.next(); v != nil; v = ww.next() {
			if _, err := sw.WriteString(fmt.Sprintf("%s\n", v.string())); err != nil {
				return err
			}
		}
		if _, err := sw.WriteString("\n"); err != nil {
			return err
		}
	}

	if len(u.usages) > 0 {
		dt := newDescriptiveTable("Usage:", u.usages)
		if err := dt.write(sw, usageColumnsWidthFactor); err != nil {
			return err
		}
	}

	if u.options.hasOptions() {
		options := make([]descriptedItem, len(u.options.allOptions))
		for i, o := range u.options.allOptions {
			desc := o.Description()
			switch o.(type) {
			case *Argumented:
				ao := o.(*Argumented)
				def := ao.Default()
				if def != "" {
					desc = fmt.Sprintf("%s Default: %s.", desc, def)
				}
			}
			options[i] = *newDescriptedItem(o.DescriptiveName(), desc)
		}

		dt := newDescriptiveTable("Options:", options)
		if err := dt.write(sw, optionsColumnsWidthFactor); err != nil {
			return err
		}
	}

	return nil
}

type descriptedItem struct {
	item        string
	description string
}

func newDescriptedItem(item, description string) *descriptedItem {
	return &descriptedItem{
		item:        item,
		description: description,
	}
}

type descriptiveTable struct {
	name         string
	items        []string
	descriptions []string
}

func newDescriptiveTable(name string, items []descriptedItem) *descriptiveTable {
	dt := descriptiveTable{
		name:         name,
		items:        make([]string, 0, len(items)),
		descriptions: make([]string, 0, len(items)),
	}
	for _, itm := range items {
		dt.items = append(dt.items, itm.item)
		dt.descriptions = append(dt.descriptions, itm.description)
	}
	return &dt
}

func (d *descriptiveTable) write(sw io.StringWriter, targetColumnsFactor float32) error {
	if _, err := sw.WriteString(fmt.Sprintf("%s\n", d.name)); err != nil {
		return err
	}

	itemsLines := make([][]string, len(d.items))
	descsLines := make([][]string, len(d.descriptions))

	targetMaxItemWidth := int((float32(screenWidth) * targetColumnsFactor)) -
		columnSpacing -
		len(twoWs) - // in case of
		len(nextLine) // multiline

	maxItemWidth := 0
	isMultilineItems := false

	for i, itm := range d.items {
		itemLines := make([]string, 0, len(d.items))
		ww, err := newWordWrapper([]rune(itm), targetMaxItemWidth)
		if err != nil {
			return err
		}
		wrappedLines := ww.strings()
		isMultilineItem := len(wrappedLines) > 1

		for j, line := range wrappedLines {
			if isMultilineItem {
				if j > 0 {
					line = fmt.Sprintf("%s%s", twoWs, line)
				}
				if j < len(wrappedLines)-1 {
					line = fmt.Sprintf("%s%s", line, nextLine)
				}
			}
			itemLines = append(itemLines, line)
			if len(line) > maxItemWidth {
				maxItemWidth = len(line)
			}
		}
		itemsLines[i] = itemLines
		isMultilineItems = isMultilineItems || isMultilineItem
	}

	descriptionWidth := screenWidth -
		columnSpacing -
		maxItemWidth -
		columnSpacing

	for i, dsc := range d.descriptions {
		ww, err := newWordWrapper([]rune(dsc), descriptionWidth)
		if err != nil {
			return err
		}
		descsLines[i] = ww.strings()
	}

	for i, itemLines := range itemsLines {
		if isMultilineItems && i > 0 {
			if _, err := sw.WriteString("\n"); err != nil {
				return err
			}
		}
		descLines := descsLines[i]

		both := len(itemLines)
		if both > len(descLines) {
			both = len(descLines)
		}

		for j := 0; j < both; j++ {
			if _, err := sw.WriteString(columnSpace); err != nil {
				return err
			}
			line := itemLines[j]
			if _, err := sw.WriteString(line); err != nil {
				return err
			}
			for k := 0; k < maxItemWidth-len(line); k++ {
				if _, err := sw.WriteString(" "); err != nil {
					return err
				}
			}
			if _, err := sw.WriteString(columnSpace); err != nil {
				return err
			}
			if _, err := sw.WriteString(descLines[j]); err != nil {
				return err
			}
			if _, err := sw.WriteString("\n"); err != nil {
				return err
			}
		}

		for j := both; j < len(itemLines); j++ {
			if _, err := sw.WriteString(columnSpace); err != nil {
				return err
			}
			if _, err := sw.WriteString(itemLines[j]); err != nil {
				return err
			}
			if _, err := sw.WriteString("\n"); err != nil {
				return err
			}
		}

		for j := both; j < len(descLines); j++ {
			if _, err := sw.WriteString(columnSpace); err != nil {
				return err
			}
			for k := 0; k < maxItemWidth; k++ {
				if _, err := sw.WriteString(" "); err != nil {
					return err
				}
			}
			if _, err := sw.WriteString(columnSpace); err != nil {
				return err
			}
			if _, err := sw.WriteString(descLines[j]); err != nil {
				return err
			}
			if _, err := sw.WriteString("\n"); err != nil {
				return err
			}
		}
	}
	return nil
}

type wordWrapper struct {
	text       []rune
	width      int
	startIndex int
	endIndex   int
}

func newWordWrapper(text []rune, width int) (w *wordWrapper, err error) {
	if width < 1 {
		return nil, errors.New("width should be 1 or more")
	}
	return &wordWrapper{
		text:       text,
		width:      width,
		startIndex: -1,
		endIndex:   -1,
	}, nil
}

func (w *wordWrapper) next() *wordWrapper {
	if w.endIndex+1 == len(w.text) {
		return nil
	}

	w.startIndex = w.endIndex + 1

	var c rune
	for {
		c = w.text[w.startIndex]
		if !unicode.IsSpace(c) {
			break
		}
		w.startIndex++
		if w.startIndex == len(w.text) {
			return nil
		}
	}

	w.endIndex = w.startIndex

	state := notWsState
	currentIndex := w.startIndex

	for {
		currentIndex++
		if currentIndex == len(w.text) {
			switch state {
			case notWsState:
				w.endIndex = currentIndex - 1
			}
			return w
		}
		c = w.text[currentIndex]
		switch state {
		case notWsState:
			if unicode.IsSpace(c) {
				w.endIndex = currentIndex - 1
				state = wsState
			}
			if currentIndex-w.startIndex+1 >= w.width {
				if w.startIndex != w.endIndex {
					return w
				}
			}
		case wsState:
			if !unicode.IsSpace(c) {
				if currentIndex-w.startIndex+1 > w.width {
					return w
				}
				state = notWsState
			}
		}
	}
}

func (w *wordWrapper) string() string {
	if w.startIndex == -1 {
		return string(w.text)
	}
	return string(w.text[w.startIndex : w.endIndex+1])
}

func (w *wordWrapper) strings() []string {
	ss := make([]string, 0, 2)
	for v := w.next(); v != nil; v = w.next() {
		ss = append(ss, v.string())
	}
	return ss
}

func (w *wordWrapper) length() int {
	if w.startIndex == -1 {
		return len(w.text)
	}
	return w.endIndex - w.startIndex + 1
}
