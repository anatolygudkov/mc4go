// Copyright (c) 2020 anatolygudkov. All rights reserved.
// Use of this source code is governed by MIT license
// that can be found in the LICENSE file.
package cli

import (
	"fmt"
	"os"
	"path/filepath"
)

// App is the main structure of a command line application.
type App struct {
	options  *Options
	help     *Flag
	question *Flag
	usage    *Usage
}

// NewApp creates new instance of the App.
// This function uses the name of the executable file without its extention as the name of the App.
func NewApp() (a *App, err error) {
	name := filepath.Base(os.Args[0])
	ext := filepath.Ext(name)
	if ext != "" {
		name = name[:(len(name) - len(ext))]
	}
	return NewNamedApp(name)
}

// NewNamedApp creates new instance of the App with the name specified.
func NewNamedApp(name string) (a *App, err error) {
	opts := NewOptions()

	usage, err := NewUsage(name, opts)
	if err != nil {
		return nil, err
	}

	a = &App{
		options:  opts,
		help:     nil,
		question: nil,
		usage:    usage,
	}

	return a, nil
}

// ExitIfError checks if the error passed isn't nil,
// writes the error's message into Stderr stream and does os.Exit with the error code 1.
func ExitIfError(err error) {
	if err == nil {
		return
	}
	os.Stderr.WriteString(err.Error())
	os.Exit(1)
}

// NewLongFlag adds new flag option with a long name specified.
func (a *App) NewLongFlag(longName string) (f *Flag, err error) {
	return a.options.NewLongFlag(longName)
}

// NewShortFlag adds new flag option with a short name specified.
func (a *App) NewShortFlag(shortName rune) (f *Flag, err error) {
	return a.options.NewShortFlag(shortName)
}

// NewFlag adds new flag option with both long and short names specified.
func (a *App) NewFlag(longName string, shortName rune) (f *Flag, err error) {
	return a.options.NewFlag(longName, shortName)
}

// NewLongArgumented adds new option with an argument with a long name specified.
func (a *App) NewLongArgumented(longName, argumentName string) (ar *Argumented, err error) {
	return a.options.NewLongArgumented(longName, argumentName)
}

// NewShortArgumented adds new option with an argument with a short name specified.
func (a *App) NewShortArgumented(shortName rune, argumentName string) (ar *Argumented, err error) {
	return a.options.NewShortArgumented(shortName, argumentName)
}

// NewArgumented adds new option with an argument with both long and short names specified.
func (a *App) NewArgumented(longName string, shortName rune, argumentName string) (ar *Argumented, err error) {
	return a.options.NewArgumented(longName, shortName, argumentName)
}

func (a *App) AddUsage(arguments, description string) {
	a.usage.AddUsage(arguments, description)
}

func (a *App) SetVersion(version string) {
	a.usage.SetVersion(version)
}

func (a *App) SetDescription(description string) {
	a.usage.SetDescription(description)
}

func (a *App) Start(work func(parameters []string) error) {
	if a.help == nil {
		help, _ := a.options.NewFlag("help", 'h')
		help.SetDescription("This help.")
		a.help = help
	}
	if a.question == nil {
		question, _ := a.options.NewShortFlag('?')
		question.SetDescription(a.help.Description())
		a.question = question
	}

	args := os.Args
	var parameters []string
	var err error
	if len(args) > 0 {
		parameters, err = a.options.Parse(os.Args[1:])
		if err != nil {
			if a.help.IsSet() || a.question.IsSet() {
				a.printHelp()
				return
			}
			os.Stderr.WriteString(fmt.Sprintf("Error: %v\n", err))
			a.printHelp()
			return
		}

		if a.help.IsSet() || a.question.IsSet() {
			a.printHelp()
			return
		}
	}

	defer func() {
		if r := recover(); r != nil {
			switch x := r.(type) {
			case error:
				err = x
			default:
				err = fmt.Errorf("panic %v", r)
			}
		}
	}()
	err = work(parameters)
	if err != nil {
		os.Stderr.WriteString(fmt.Sprintf("Error: %v\n", err))
	}
}

func (a *App) printHelp() {
	a.usage.Write(os.Stdout)
}
