// Copyright (c) 2020 anatolygudkov. All rights reserved.
// Use of this source code is governed by MIT license
// that can be found in the LICENSE file.
package cli

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode"
)

const (
	paramExpectedState    = 0
	argumentExpectedState = 1
)

type optionInfo interface {
	LongName() string
	ShortName() rune
	DescriptiveName() string
	Description() string
	IsRequired() bool
}

// Options allows to define flags and options with arguments in getopt_long style.
type Options struct {
	longOptions  map[string]optionInfo
	shortOptions map[rune]optionInfo
	allOptions   []optionInfo
	arguments    map[string]*string // Key is option's descriptive name
	parsed       bool
}

// NewOptions creates a new instance of Options
func NewOptions() *Options {
	return &Options{
		longOptions:  make(map[string]optionInfo),
		shortOptions: make(map[rune]optionInfo),
		allOptions:   make([]optionInfo, 0),
		arguments:    make(map[string]*string),
		parsed:       false,
	}
}

// NewLongFlag adds new flag option with a long name specified.
func (opts *Options) NewLongFlag(longName string) (f *Flag, err error) {
	return opts.NewFlag(longName, 0)
}

// NewShortFlag adds new flag option with a short name specified.
func (opts *Options) NewShortFlag(shortName rune) (f *Flag, err error) {
	return opts.NewFlag("", shortName)
}

// NewFlag adds new flag option with both long and short names specified.
func (opts *Options) NewFlag(longName string, shortName rune) (f *Flag, err error) {
	f = new(Flag)

	f.owner = opts

	if err = f.setLongShortNames(longName, shortName); err != nil {
		return nil, err
	}

	var dn strings.Builder
	if shortName != 0 {
		dn.WriteRune('-')
		dn.WriteRune(shortName)
	}
	if longName != "" {
		if dn.Len() > 0 {
			dn.WriteString("  or  ")
		}
		dn.WriteString("--")
		dn.WriteString(longName)
	}
	f.descriptiveName = dn.String()

	if err := opts.registerOption(f); err != nil {
		return nil, err
	}

	return f, err
}

// NewLongArgumented adds new option with an argument with a long name specified.
func (opts *Options) NewLongArgumented(longName string, argumentName string) (a *Argumented, err error) {
	return opts.NewArgumented(longName, 0, argumentName)
}

// NewShortArgumented adds new option with an argument with a short name specified.
func (opts *Options) NewShortArgumented(shortName rune, argumentName string) (a *Argumented, err error) {
	return opts.NewArgumented("", shortName, argumentName)
}

// NewArgumented adds new option with an argument with both long and short names specified.
func (opts *Options) NewArgumented(longName string, shortName rune, argumentName string) (a *Argumented, err error) {
	a = new(Argumented)

	a.owner = opts

	if err = a.setLongShortNames(longName, shortName); err != nil {
		return nil, err
	}

	var dn strings.Builder
	if shortName != 0 {
		dn.WriteRune('-')
		dn.WriteRune(shortName)
		dn.WriteString(" <")
		dn.WriteString(argumentName)
		dn.WriteRune('>')
	}
	if longName != "" {
		if dn.Len() > 0 {
			dn.WriteString("  or  ")
		}
		dn.WriteString("--")
		dn.WriteString(longName)
		dn.WriteString(" <")
		dn.WriteString(argumentName)
		dn.WriteRune('>')
	}
	a.descriptiveName = dn.String()

	a.argumentName = argumentName

	if err := opts.registerOption(a); err != nil {
		return nil, err
	}

	return a, nil
}

// Parse parses command line arguments to set found flags and options' arguments.
// It returns remaining program parameters and an error if happened while parsing.
// Passed args shouldn't start with the name of the executable.
func (opts *Options) Parse(args []string) (parameters []string, err error) {
	opts.parsed = true

	if len(opts.arguments) > 0 {
		opts.arguments = make(map[string]*string)
	}

	parameters = make([]string, 0, len(args))

	currentIndex := 0

	state := paramExpectedState
	var currentOptionToArgument *Argumented = nil
Loop:
	for currentIndex < len(args) {
		s := args[currentIndex]
		s = strings.TrimSpace(s)
		if s == "" {
			currentIndex++
			continue
		}

		rs := []rune(s)
		switch firstChar := rs[0]; firstChar {
		case '-':
			switch state {
			case paramExpectedState:
				if len(rs) == 1 {
					return nil, errors.New("'-' isn't allowed option")
				}
				switch secondChar := s[1]; secondChar {
				case '-':
					if len(rs) == 2 { // '--' - end of the options
						currentIndex++
						break Loop
					}
					if currentOptionToArgument, err = opts.parseLong(rs); err != nil {
						return nil, err
					}
					if currentOptionToArgument != nil {
						state = argumentExpectedState
					}
				default:
					if currentOptionToArgument, err = opts.parseShort(rs); err != nil {
						return nil, err
					}
					if currentOptionToArgument != nil {
						state = argumentExpectedState
					}
				}
			case argumentExpectedState:
				return nil, fmt.Errorf("no argument found for the option: %s", currentOptionToArgument.descriptiveName)
			default:
				return nil, errors.New("unexpected internal state")
			}
		default:
			switch state {
			case paramExpectedState:
				parameters = append(parameters, s)
			case argumentExpectedState:
				opts.arguments[currentOptionToArgument.DescriptiveName()] = &s
				currentOptionToArgument = nil
				state = paramExpectedState
			default:
				return nil, errors.New("unexpected internal state")
			}
		}
		currentIndex++
	}

	if state == argumentExpectedState {
		return nil, fmt.Errorf("no required arg found for the option: %s", currentOptionToArgument.longName)
	}

	// Validate required options
	var missedRequires strings.Builder
	var missed = 0
	for _, o := range opts.allOptions {
		if _, has := opts.arguments[o.DescriptiveName()]; !o.IsRequired() || has {
			continue
		}
		if missedRequires.Len() > 0 {
			missedRequires.WriteString(", ")
		}
		missedRequires.WriteString(fmt.Sprintf("'%s'", o.DescriptiveName()))
		missed++
	}
	if missed > 0 {
		var opts = "option"
		if missed > 1 {
			opts = fmt.Sprintf("%ss", opts)
		}
		return nil, fmt.Errorf("Required %s missed: %s", opts, missedRequires.String())
	}

	for _, o := range opts.allOptions {
		if !o.IsRequired() {
			continue
		}
	}

	for i := currentIndex; i < len(args); i++ {
		parameters = append(parameters, args[i])
	}

	return parameters, nil
}

func (opts *Options) parseShort(rs []rune) (o *Argumented, err error) {
	var argument strings.Builder

	for i := 1; i < len(rs); i++ { // We know that 'rs' consists of at least 2 chars
		c := rs[i]

		if o != nil {
			argument.WriteRune(c)
			continue
		}

		nextOption, has := opts.shortOptions[c]
		if !has {
			var msg string = fmt.Sprintf("unknown option '-%c'", c)
			if len(rs) > 2 {
				msg = fmt.Sprintf("%s in '%s'", msg, string(rs))
			}
			return nil, errors.New(msg)
		}

		switch nextOption.(type) {
		case *Argumented:
			o = nextOption.(*Argumented)
			continue
		default:
			o = nil
		}

		if _, has := opts.arguments[nextOption.DescriptiveName()]; has {
			return nil, fmt.Errorf("option '%s' is duplicated in '%s'", nextOption.DescriptiveName(), string(rs))
		}
		opts.arguments[nextOption.DescriptiveName()] = nil
	}

	if o == nil {
		return o, nil
	}

	if argument.Len() > 0 {
		s := argument.String()
		opts.arguments[o.DescriptiveName()] = &s
		o = nil
		return o, nil
	}

	return o, nil
}

func (opts *Options) parseLong(rs []rune) (o *Argumented, err error) {
	var name strings.Builder
	var argument *strings.Builder = nil

	for i := 2; i < len(rs); i++ {
		c := rs[i]
		if argument != nil {
			argument.WriteRune(c)
			continue
		}
		if c == '=' {
			argument = new(strings.Builder)
			continue
		}
		if unicode.IsLetter(c) || unicode.IsDigit(c) {
			name.WriteRune(c)
			continue
		}
		return nil, fmt.Errorf("wrong character '%c' in the long name: %s", c, string(rs))
	}

	longName := name.String()

	oi, has := opts.longOptions[longName]
	if !has {
		return nil, fmt.Errorf("unknown option '--%s'", longName)
	}

	if _, has := opts.arguments[oi.DescriptiveName()]; has {
		return nil, fmt.Errorf("option '%s' duplicated in '%s'", oi.DescriptiveName(), string(rs))
	}

	opts.arguments[oi.DescriptiveName()] = nil

	switch oi.(type) {
	case *Argumented:
		o = oi.(*Argumented)
	default:
		if argument != nil {
			return nil, fmt.Errorf("option %s is a flag and cannot have an argument", oi.DescriptiveName())
		}
		return o, nil
	}

	if argument != nil {
		if argument.Len() == 0 {
			return nil, fmt.Errorf("option %s is a flag and cannot have an argument", oi.DescriptiveName())
		}
		s := argument.String()
		opts.arguments[o.DescriptiveName()] = &s
		o = nil
	}

	return o, nil
}

func (opts *Options) registerOption(oi optionInfo) (err error) {
	if oi.LongName() != "" {
		if _, has := opts.longOptions[oi.LongName()]; has {
			return fmt.Errorf("duplicated long option: %s", oi.LongName())
		}
		opts.longOptions[oi.LongName()] = oi
	}
	if oi.ShortName() != 0 {
		if _, has := opts.shortOptions[oi.ShortName()]; has {
			return fmt.Errorf("duplicated short option: %c", oi.ShortName())
		}
		opts.shortOptions[oi.ShortName()] = oi
	}

	opts.allOptions = append(opts.allOptions, oi)

	return nil
}

func (opts *Options) hasOptions() bool {
	return len(opts.allOptions) > 0
}

// Option presents the contract common for both a flag and an option with an argument.
type Option struct {
	owner           *Options
	longName        string
	shortName       rune
	descriptiveName string
	description     string
	required        bool
}

// LongName returns the long name of the option.
func (o *Option) LongName() string {
	return o.longName
}

// ShortName returns the short name of the option.
func (o *Option) ShortName() rune {
	return o.shortName
}

// DescriptiveName returns the descriptive name of the option. Typically it consists of both long and short names.
func (o *Option) DescriptiveName() string {
	return o.descriptiveName
}

// Description returns the description of the option.
func (o *Option) Description() string {
	return o.description
}

// SetDescription sets the description of the option.
func (o *Option) SetDescription(d string) {
	o.description = d
}

// IsRequired returns true if the option is required, otherwise it return false.
func (o *Option) IsRequired() bool {
	return o.required
}

// Require makes the option required.
func (o *Option) Require() {
	o.required = true
}

// IsSet returns true if the option was recognized as a set one while parsing.
func (o *Option) IsSet() bool {
	_, has := o.owner.arguments[o.DescriptiveName()]
	return has
}

func (o *Option) setLongShortNames(longName string, shortName rune) (err error) {
	if longName == "" &&
		shortName == 0 {
		return errors.New("long name or short name should be specified")
	}

	o.longName = longName
	o.shortName = shortName

	return nil
}

// Flag presents a flag option.
type Flag struct {
	Option
}

// Argumented presents an option with an argument.
type Argumented struct {
	Option
	argumentName         string
	defaultArgumentValue string
}

// Require makes the option with an argument required.
func (a *Argumented) Require() {
	a.Option.Require()
	a.defaultArgumentValue = ""
}

// ArgumentName returns name of the argument of the option.
func (a *Argumented) ArgumentName() (s string) {
	return a.argumentName
}

// SetDefault sets the default value of the option.
func (a *Argumented) SetDefault(s string) {
	a.required = false
	a.defaultArgumentValue = s
}

// Default returns the default value of the option.
func (a *Argumented) Default() (s string) {
	return a.defaultArgumentValue
}

// String returns a string value of the option if available after parsing. ok is false if no value available.
func (a *Argumented) String() (s string, ok bool) {
	if !a.owner.parsed {
		return "", false
	}
	v := a.owner.arguments[a.DescriptiveName()]
	if v == nil {
		if a.defaultArgumentValue != "" {
			return a.defaultArgumentValue, true
		}
		return "", false
	}
	return *v, true
}

// Int returns an int value of the option if available after parsing. ok is false if no value available.
func (a *Argumented) Int() (i int, ok bool, err error) {
	s, ok := a.String()
	if !ok {
		return 0, ok, nil
	}
	i, err = strconv.Atoi(s)
	return i, ok, err
}

// Bool returns a bool value of the option if available after parsing. ok is false if no value available.
func (a *Argumented) Bool() (b bool, ok bool) {
	s, ok := a.String()
	if !ok {
		return false, ok
	}

	s = strings.ToLower(strings.TrimSpace(s))
	return s == "y" ||
		s == "yes" ||
		s == "true" ||
		s == "1", true
}

// FileInfo returns a FileInfo using string value of the option if available after parsing. ok is false if no value available.
func (a *Argumented) FileInfo() (f os.FileInfo, ok bool, err error) {
	s, ok := a.String()
	if !ok {
		return nil, ok, nil
	}
	f, err = os.Stat(s)
	return f, ok, err
}

// ExistingFileInfo returns a FileInfo using string value of the option after parsing.
// It returns an error if the file doesn't exist.
func (a *Argumented) ExistingFileInfo() (f os.FileInfo, err error) {
	s, ok := a.String()
	if !ok {
		return nil, fmt.Errorf("no value for the option %s specified", a.DescriptiveName())
	}
	f, err = os.Stat(s)
	return f, err
}
