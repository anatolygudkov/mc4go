// Copyright (c) 2020 anatolygudkov. All rights reserved.
// Use of this source code is governed by MIT license
// that can be found in the LICENSE file.
package mc4go

import (
	"fmt"
	"os"
	"path"
	"sync"
	"testing"
)

const (
	propertyPrefix = "property"
	valuePrefix    = "value"
	counterPrefix  = "counter"
)

func TestFull(t *testing.T) {
	numberOfStatics := 1000
	numberOfCounters := 1000

	filename := path.Join(GetMCountersDirectoryPath(), "goTestFull.dat")
	_, err := os.Stat(filename)
	if err == nil {
		if os.Remove(filename) != nil {
			t.Fatal(err)
		}
	}

	statics := make(map[string]string)
	for i := 0; i < numberOfStatics; i++ {
		statics[fmt.Sprintf("%s%d", propertyPrefix, i)] = fmt.Sprintf("%s%d", valuePrefix, i)
	}

	writer, err := NewWriterForFile(filename, statics, numberOfCounters)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(filename)
	defer writer.Close()

	reader, err := NewReaderForFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	var counters []*Counter

	for i := 0; i < numberOfCounters; i++ {
		counter, err := writer.AddCounter(fmt.Sprintf("%s%d", counterPrefix, i))
		if err != nil {
			t.Fatal(err)
		}
		counters = append(counters, counter)

		counter.Set(counter.id - 1)
		counter.Increment()
	}

	numOfStaticsCounted := 0

	reader.ForEachStatic(func(label, value string) bool {
		staticLabel := fmt.Sprintf("%s%d", propertyPrefix, numOfStaticsCounted)

		expectedValue := fmt.Sprintf("%s%d", valuePrefix, numOfStaticsCounted)
		readValue := statics[staticLabel]

		if readValue != expectedValue {
			t.Fatalf("Read static value %s, expected %s", readValue, expectedValue)
		}

		gotValue, err := reader.GetStaticValue(staticLabel)
		if err != nil {
			t.Fatal(err)
		}
		if readValue != expectedValue {
			t.Fatalf("Got static value %s, expected %s", gotValue, expectedValue)
		}

		delete(statics, staticLabel)

		numOfStaticsCounted++

		return true
	})
	if numOfStaticsCounted != numberOfStatics {
		t.Fatalf("Statics counted %d, expected %d", numOfStaticsCounted, numberOfStatics)
	}
	if len(statics) != 0 {
		t.Fatal("All statics must be removed")
	}

	numOfCountersCounted := 0

	reader.ForEachCounter(func(id, value int64, label string) bool {
		counterLabel := fmt.Sprintf("%s%d", counterPrefix, value)

		if label != counterLabel {
			t.Fatalf("Got counter label %s, expected %s", label, counterLabel)
		}

		foundLabel, err := reader.GetCounterLabel(id)
		if err != nil {
			t.Fatal(err)
		}

		if foundLabel != label {
			t.Fatalf("Found counter label %s, expected %s", foundLabel, label)
		}

		numOfCountersCounted++

		return true
	})
	if numOfCountersCounted != numberOfCounters {
		t.Fatalf("Counters counted %d, expected %d", numOfCountersCounted, numberOfCounters)
	}

	for _, c := range counters {
		if c.IsClosed() {
			t.Error("The counter must not be closed")
		}
		c.Close()
		if !c.IsClosed() {
			t.Fatal("The counter must be closed")
		}
	}

	reader.ForEachCounter(func(id, value int64, label string) bool {
		t.Error("No counters must be available")
		return true
	})
}

func TestConcurrentCountersModification(t *testing.T) {
	numberOfCounters := 2

	filename := path.Join(GetMCountersDirectoryPath(), "goTestConcurrentCountersModification.dat")
	_, err := os.Stat(filename)
	if err == nil {
		if os.Remove(filename) != nil {
			t.Error(err)
		}
	}

	statics := make(map[string]string)

	writer, err := NewWriterForFile(filename, statics, numberOfCounters)
	if err != nil {
		t.Error(err)
	}
	defer os.Remove(filename)
	defer writer.Close()

	reader, err := NewReaderForFile(filename)
	if err != nil {
		t.Error(err)
	}
	defer reader.Close()

	cnt0, err := writer.AddCounterWithInitialValue(fmt.Sprintf("%s%d", counterPrefix, 0), 0)
	if err != nil {
		t.Error(err)
	}
	cnt1, err := writer.AddCounterWithInitialValue(fmt.Sprintf("%s%d", counterPrefix, 1), 1)
	if err != nil {
		t.Error(err)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	var lastValue int64 = 2_000_000

	ping := func() {
		currentValue0 := cnt0.Get()
		currentValue1 := cnt1.Get()

		for currentValue0 < lastValue {
			if currentValue1 > currentValue0 {
				cnt0.Set(currentValue1)
			}
			currentValue0 = cnt0.Get()
			currentValue1 = cnt1.Get()
		}

		if lastValue != currentValue0 {
			t.Error("Ping failed")
		}

		wg.Done()
	}

	pong := func() {
		currentValue0 := cnt0.Get()
		currentValue1 := cnt1.Get()

		for currentValue0 < lastValue {
			if currentValue1 == currentValue0 {
				cnt1.Set(currentValue1 + 1)
			}
			currentValue0 = cnt0.Get()
			currentValue1 = cnt1.Get()
		}

		if lastValue != currentValue0 {
			t.Error("Pong failed")
		}

		wg.Done()
	}

	go ping()
	go pong()

	wg.Wait()
}

func TestConcurrentCountersAddClose(t *testing.T) {
	numberOfCounters := 5

	filename := path.Join(GetMCountersDirectoryPath(), "goTestConcurrentCountersAddClose.dat")
	_, err := os.Stat(filename)
	if err == nil {
		if os.Remove(filename) != nil {
			t.Error(err)
		}
	}

	statics := make(map[string]string)

	writer, err := NewWriterForFile(filename, statics, numberOfCounters)
	if err != nil {
		t.Error(err)
	}
	defer os.Remove(filename)
	defer writer.Close()

	reader, err := NewReaderForFile(filename)
	if err != nil {
		t.Error(err)
	}
	defer reader.Close()

	var wg sync.WaitGroup
	wg.Add(2)

	iterations := 1_000_000

	gen0 := func() {
		for i := 0; i < iterations; i++ {
			addAndCloseCounter(t, writer, reader, i)
		}
		wg.Done()
	}

	gen1 := func() {
		for i := 0; i < iterations; i++ {
			addAndCloseCounter(t, writer, reader, i)
		}
		wg.Done()
	}

	go gen0()
	go gen1()

	wg.Wait()
}

func addAndCloseCounter(t *testing.T, w *Writer, r *Reader, i int) {
	cnt, err := w.AddCounterWithInitialValue(fmt.Sprintf("%s%d", counterPrefix, i), int64(i))
	if err != nil {
		t.Error(err)
	}

	label, err := r.GetCounterLabel(cnt.ID())
	if err != nil {
		t.Error(err)
	}
	if label != cnt.Label() {
		t.Error("Fail")
	}

	value, err := r.GetCounterValue(cnt.ID())
	if err != nil {
		t.Error(err)
	}
	if value != cnt.Get() {
		t.Error("Fail")
	}

	cnt.Close()

	label, err = r.GetCounterLabel(cnt.ID())
	if err == nil {
		t.Error("Fail")
	}

	value, err = r.GetCounterValue(cnt.ID())
	if err == nil {
		t.Error("Fail")
	}
}
