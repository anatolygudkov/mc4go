// Copyright (c) 2020 anatolygudkov. All rights reserved.
// Use of this source code is governed by MIT license
// that can be found in the LICENSE file.
package main

import (
	"fmt"

	"github.com/anatolygudkov/mc4go"
	"github.com/anatolygudkov/mc4go/internal/app/cli"
)

func main() {
	a, err := cli.NewApp()
	cli.ExitIfError(err)

	fileArg, err := a.NewArgumented("file", 'f', "FILE")
	cli.ExitIfError(err)

	fileArg.SetDescription("Path to a counters' file to be parsed.")
	fileArg.Require()

	a.AddUsage("--file /dev/shm/jmx_counters.dat", "Prints content of the /dev/shm/jmx_counters.dat file.")

	a.Start(func(parameters []string) error {
		file, _ := fileArg.String() //Must have value, since required

		fmt.Printf("file: %s\n", file)

		r, err := mc4go.NewReaderForFile(file)
		if err != nil {
			return err
		}
		defer r.Close()

		fmt.Printf("version: %d\n", r.Version())
		fmt.Printf("pid: %d\n", r.Pid())
		fmt.Printf("started: %d\n", r.StartTime())

		r.ForEachStatic(func(label, value string) bool {
			fmt.Printf("static: %s=%s\n", label, value)
			return true
		})

		r.ForEachCounter(func(id, value int64, label string) bool {
			fmt.Printf("counter: %s[%d]=%d\n", label, id, value)
			return true
		})

		return nil
	})
}
