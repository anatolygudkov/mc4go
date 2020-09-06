// Copyright (c) 2020 anatolygudkov. All rights reserved.
// Use of this source code is governed by MIT license
// that can be found in the LICENSE file.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/anatolygudkov/mc4go"
	"github.com/anatolygudkov/mc4go/internal/app/cli"
	"github.com/anatolygudkov/mc4go/internal/app/rest"
)

type Dump struct {
	File     string    `json:"file"`
	Version  int32     `json:"version"`
	Pid      int64     `json:"pid"`
	Started  int64     `json:"started"`
	Statics  []Static  `json:"statics"`
	Counters []Counter `json:"counters"`
}

type Statics struct {
	Statics []Static `json:"statics"`
}

type Static struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type Counters struct {
	Counters []Counter `json:"counters"`
}

type Counter struct {
	ID    int64  `json:"id"`
	Label string `json:"label"`
	Value int64  `json:"value"`
}

func collectStatics(r *mc4go.Reader) (s []Static) {
	r.ForEachStatic(func(lbl, val string) bool {
		s = append(s, Static{Label: lbl, Value: val})
		return true
	})
	return s
}

func collectCounters(r *mc4go.Reader) (c []Counter) {
	r.ForEachCounter(func(id, val int64, lbl string) bool {
		c = append(c, Counter{ID: id, Value: val, Label: lbl})
		return true
	})
	return c
}

func answerJSON(res http.ResponseWriter, v interface{}) error {
	res.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(res).Encode(v)
}

func doFile(values *rest.Values, res http.ResponseWriter, req *http.Request, r *mc4go.Reader, file string) error {
	return answerJSON(res, file)
}

func doVersion(values *rest.Values, res http.ResponseWriter, req *http.Request, r *mc4go.Reader, file string) error {
	return answerJSON(res, r.Version())
}

func doPid(values *rest.Values, res http.ResponseWriter, req *http.Request, r *mc4go.Reader, file string) error {
	return answerJSON(res, r.Pid())
}

func doStarted(values *rest.Values, res http.ResponseWriter, req *http.Request, r *mc4go.Reader, file string) error {
	return answerJSON(res, r.StartTime())
}

func doDump(values *rest.Values, res http.ResponseWriter, req *http.Request, r *mc4go.Reader, file string) error {
	d := new(Dump)
	d.File = file
	d.Version = r.Version()
	d.Pid = r.Pid()
	d.Started = r.StartTime()
	d.Statics = collectStatics(r)
	d.Counters = collectCounters(r)
	return answerJSON(res, d)
}

func doStatic(values *rest.Values, res http.ResponseWriter, req *http.Request, r *mc4go.Reader) error {
	l := values.String("label")
	if l == "" {
		return errors.New("label isn't specified")
	}
	v, err := r.GetStaticValue(l)
	if err != nil {
		return fmt.Errorf("cannot find a static with the label: '%s'", l)
	}
	return answerJSON(res, v)
}

func doStatics(values *rest.Values, res http.ResponseWriter, req *http.Request, r *mc4go.Reader) error {
	s := new(Statics)
	s.Statics = collectStatics(r)
	return answerJSON(res, s)
}

func doCounter(values *rest.Values, res http.ResponseWriter, req *http.Request, r *mc4go.Reader) error {
	il := values.String("id_label")
	if il == "" {
		return errors.New("not id nor label specified")
	}
	if id, err := strconv.Atoi(il); err == nil {
		v, err := r.GetCounterValue(int64(id))
		if err != nil {
			return err
		}
		return answerJSON(res, v)
	}
	var v int64
	found := false
	r.ForEachCounter(
		func(id, value int64, label string) bool {
			if label == il {
				v = value
				found = true
				return false
			}
			return true
		})
	if !found {
		return fmt.Errorf("no counter with the label '%s' found", il)
	}
	return answerJSON(res, v)
}

func doCounters(values *rest.Values, res http.ResponseWriter, req *http.Request, r *mc4go.Reader) error {
	c := new(Counters)
	c.Counters = collectCounters(r)
	return answerJSON(res, c)
}

func main() {
	a, err := cli.NewApp()
	cli.ExitIfError(err)

	fileArg, err := a.NewArgumented("file", 'f', "FILE")
	cli.ExitIfError(err)

	fileArg.SetDescription("Path to a counters' file to be parsed.")
	fileArg.Require()

	addrArg, err := a.NewArgumented("addr", 'a', "ADDR")
	cli.ExitIfError(err)
	addrArg.SetDescription("Local address to listen to the incoming requests. For example: 192.168.1.12:8000, :8888.")
	addrArg.SetDefault("127.0.0.1:8888")

	a.AddUsage("--file /dev/shm/jmx_counters.dat", "Exposes content of the /dev/shm/jmx_counters.dat file.")

	a.Start(func(parameters []string) error {
		addr, _ := addrArg.String() // Must have a value, since has a default one

		file, _ := fileArg.String() //Must have value, since required

		r, err := mc4go.NewReaderForFile(file)
		cli.ExitIfError(err)
		defer r.Close()

		srv := rest.NewSrv(addr)

		srv.Get("/dump", func(values *rest.Values, res http.ResponseWriter, req *http.Request) error {
			return doDump(values, res, req, r, file)
		})
		srv.Get("/file", func(values *rest.Values, res http.ResponseWriter, req *http.Request) error {
			return doFile(values, res, req, r, file)
		})
		srv.Get("/version", func(values *rest.Values, res http.ResponseWriter, req *http.Request) error {
			return doVersion(values, res, req, r, file)
		})
		srv.Get("/pid", func(values *rest.Values, res http.ResponseWriter, req *http.Request) error {
			return doPid(values, res, req, r, file)
		})
		srv.Get("/started", func(values *rest.Values, res http.ResponseWriter, req *http.Request) error {
			return doStarted(values, res, req, r, file)
		})
		srv.Get("/static/:label", func(values *rest.Values, res http.ResponseWriter, req *http.Request) error {
			return doStatic(values, res, req, r)
		})
		srv.Get("/statics", func(values *rest.Values, res http.ResponseWriter, req *http.Request) error {
			return doStatics(values, res, req, r)
		})
		srv.Get("/counter/:id_label", func(values *rest.Values, res http.ResponseWriter, req *http.Request) error {
			return doCounter(values, res, req, r)
		})
		srv.Get("/counters", func(values *rest.Values, res http.ResponseWriter, req *http.Request) error {
			return doCounters(values, res, req, r)
		})

		return srv.Start()
	})
}
