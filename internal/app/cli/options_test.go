// Copyright (c) 2020 anatolygudkov. All rights reserved.
// Use of this source code is governed by MIT license
// that can be found in the LICENSE file.
package cli

import (
	"fmt"
	"strings"
	"testing"
)

func TestNoOptions(t *testing.T) {
	opts := NewOptions()

	params, err := opts.Parse([]string{})
	if err != nil {
		t.Fatal(err)
	}
	if params == nil || len(params) > 0 {
		t.Fatal("Parameters should be empty")
	}

	args := []string{"param1", "param2"}
	params, err = opts.Parse(args)
	if err != nil {
		t.Fatal(err)
	}
	if params == nil || len(params) != 2 {
		t.Fatal("2 parameters should be parsed")
	}
	if params[0] != args[0] {
		t.Fatalf("Parameter 0: expected %s, got %s", params[0], args[0])
	}
	if params[1] != args[1] {
		t.Fatalf("Parameter 1: expected %s, got %s", params[1], args[1])
	}
}

func TestUnknownOption(t *testing.T) {
	opts := NewOptions()

	args := []string{"param1", "-x", "param2"}
	_, err := opts.Parse(args)
	if err == nil {
		t.Fatal("An error expected")
	}
	errStr := strings.ToLower(err.Error())
	if !strings.Contains(errStr, "unknown option '-x'") {
		t.Fatalf("%s should be unknown", args[1])
	}

	args = []string{"param1", "--xyz", "param2"}
	_, err = opts.Parse(args)
	if err == nil {
		t.Fatal("An error expected")
	}
	errStr = strings.ToLower(err.Error())
	if !strings.Contains(errStr, "unknown option '--xyz'") {
		t.Fatalf("%s should be unknown", args[1])
	}
}

func TestRequiredShortOption(t *testing.T) {
	opts := NewOptions()

	args := []string{"param1", "-x", "param2"}

	x, err := opts.NewShortFlag('x')
	if err != nil {
		t.Fatal(err)
	}
	x.Require()

	_, err = opts.Parse(args)
	if err != nil {
		t.Fatal(err)
	}

	if !x.IsSet() {
		t.Fatalf("%s should be set", x.DescriptiveName())
	}

	y, err := opts.NewShortFlag('y')
	if err != nil {
		t.Fatal(err)
	}
	y.Require()

	_, err = opts.Parse(args)
	if err == nil {
		t.Fatal("An error expected")
	}

	msg := strings.ToLower(err.Error())
	if !strings.Contains(msg, "required") ||
		!strings.Contains(msg, fmt.Sprintf("%s", y.DescriptiveName())) {
		t.Fatalf("-%s should be lost as required one", y.DescriptiveName())
	}
}

func TestRequiredLongOption(t *testing.T) {
	opts := NewOptions()

	args := []string{"param1", "--xx", "param2"}

	xx, err := opts.NewLongFlag("xx")
	if err != nil {
		t.Fatal(err)
	}
	xx.Require()

	_, err = opts.Parse(args)
	if err != nil {
		t.Fatal(err)
	}

	if !xx.IsSet() {
		t.Fatalf("%s should be set", xx.DescriptiveName())
	}

	yy, err := opts.NewLongFlag("yy")
	if err != nil {
		t.Fatal(err)
	}
	yy.Require()

	_, err = opts.Parse(args)
	if err == nil {
		t.Fatal("An error expected")
	}

	msg := strings.ToLower(err.Error())
	if !strings.Contains(msg, "required") ||
		!strings.Contains(msg, fmt.Sprintf("%s", yy.DescriptiveName())) {
		t.Fatalf("-%s should be lost as required one", yy.DescriptiveName())
	}
}

func TestDefaultShortOption(t *testing.T) {
	opts := NewOptions()

	args := []string{"param1", "-x", "valueX"}

	x, err := opts.NewShortArgumented('x', "VALUEX")
	if err != nil {
		t.Fatal(err)
	}
	x.SetDefault("DEFAULTX")
	x.Require()
	if x.Default() != "" {
		t.Fatal("Require() cleans up the default value")
	}

	_, err = opts.Parse(args)
	if err != nil {
		t.Fatal(err)
	}

	if !x.IsSet() {
		t.Fatalf("%s should be set", x.DescriptiveName())
	}
	if v, ok := x.String(); !ok || v != args[2] {
		t.Fatalf("%s should be set to %s", x.DescriptiveName(), args[2])
	}

	y, err := opts.NewShortArgumented('y', "VALUEY")
	if err != nil {
		t.Fatal(err)
	}
	y.Require()
	y.SetDefault("DEFAULTY")
	if y.IsRequired() {
		t.Fatal("Require() should be reset by the default value")
	}

	_, err = opts.Parse(args)
	if err != nil {
		t.Fatal(err)
	}
	if y.IsSet() {
		t.Fatalf("%s should not be set", y.DescriptiveName())
	}
	if v, ok := y.String(); !ok || v != y.Default() {
		t.Fatalf("%s should be set to %s", y.DescriptiveName(), y.Default())
	}
}

func TestDefaultLongOption(t *testing.T) {
	opts := NewOptions()

	args := []string{"param1", "--xx", "valueXX"}

	xx, err := opts.NewLongArgumented("xx", "VALUEXX")
	if err != nil {
		t.Fatal(err)
	}
	xx.SetDefault("DEFAULTXX")
	xx.Require()
	if xx.Default() != "" {
		t.Fatal("Require() cleans up the default value")
	}

	_, err = opts.Parse(args)
	if err != nil {
		t.Fatal(err)
	}

	if !xx.IsSet() {
		t.Fatalf("%s should be set", xx.DescriptiveName())
	}
	if v, ok := xx.String(); !ok || v != args[2] {
		t.Fatalf("%s should be set to %s", xx.DescriptiveName(), args[2])
	}

	yy, err := opts.NewLongArgumented("yy", "VALUEYY")
	if err != nil {
		t.Fatal(err)
	}
	yy.Require()
	yy.SetDefault("DEFAULTYY")
	if yy.IsRequired() {
		t.Fatal("Require() should be reset by the default value")
	}

	_, err = opts.Parse(args)
	if err != nil {
		t.Fatal(err)
	}
	if yy.IsSet() {
		t.Fatalf("%s should not be set", yy.DescriptiveName())
	}
	if v, ok := yy.String(); !ok || v != yy.Default() {
		t.Fatalf("%s should be set to %s", yy.DescriptiveName(), yy.Default())
	}
}

func TestAllOptions(t *testing.T) {
	opts := NewOptions()

	q, err := opts.NewShortFlag('q')
	if err != nil {
		t.Fatal(err)
	}

	x, err := opts.NewShortArgumented('x', "VALUEX")
	if err != nil {
		t.Fatal(err)
	}

	y, err := opts.NewFlag("single-y", 'y')
	if err != nil {
		t.Fatal(err)
	}

	z, err := opts.NewShortArgumented('z', "VALUEZ")
	if err != nil {
		t.Fatal(err)
	}

	xx, err := opts.NewLongArgumented("xx", "VALUEXX")
	if err != nil {
		t.Fatal(err)
	}
	xx.Require()

	yy, err := opts.NewLongFlag("yy")
	if err != nil {
		t.Fatal(err)
	}
	yy.Require()

	zz, err := opts.NewLongArgumented("zz", "VALUEZZ")
	if err != nil {
		t.Fatal(err)
	}
	zz.SetDefault("valZZ")

	args := []string{"-q", "-x", "valX", "-yz", "valZ", "--xx", "valXX", "--yy", "param1", "param2", "--", "--zz"}

	params, err := opts.Parse(args)
	if err != nil {
		t.Fatal(err)
	}

	if !q.IsSet() {
		t.Fatalf("%c should be set", q.ShortName())
	}

	if !x.IsSet() {
		t.Fatalf("%c should be set", x.ShortName())
	}
	if v, ok := x.String(); !ok || v != args[2] {
		t.Fatalf("%c should be set to %s", x.ShortName(), args[2])
	}

	if !y.IsSet() {
		t.Fatalf("%c should be set", y.ShortName())
	}

	if !z.IsSet() {
		t.Fatalf("%c should be set", z.ShortName())
	}
	if v, ok := z.String(); !ok || v != args[4] {
		t.Fatalf("%c should be set to %s", z.ShortName(), args[4])
	}

	if !xx.IsSet() {
		t.Fatalf("%s should be set", xx.LongName())
	}
	if v, ok := xx.String(); !ok || v != args[6] {
		t.Fatalf("%s should be set to %s", xx.LongName(), args[6])
	}

	if !yy.IsSet() {
		t.Fatalf("%s should be set", yy.LongName())
	}

	if zz.IsSet() {
		t.Fatalf("%s should not be set", zz.LongName())
	}
	if v, ok := zz.String(); !ok || v != zz.defaultArgumentValue {
		t.Fatalf("%s should be set to %s", zz.LongName(), zz.defaultArgumentValue)
	}

	if params == nil || len(params) != 3 {
		t.Fatal("3 parameters should be parsed")
	}

	if params[0] != args[8] || params[1] != args[9] || params[2] != args[11] {
		t.Fatalf("Parameters aren't parsed correctly: %v", params)
	}
}
