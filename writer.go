// Copyright (c) 2020 anatolygudkov. All rights reserved.
// Use of this source code is governed by MIT license
// that can be found in the LICENSE file.
package mc4go

import (
	"fmt"
	"os"
	"path"
	"sync/atomic"
	"time"

	"github.com/anatolygudkov/mc4go/internal/layout"
	"github.com/anatolygudkov/mc4go/internal/mmap"
	"github.com/anatolygudkov/mc4go/internal/offheap"
)

// MaxPossibleNumberOfCounters defines how many counters can exist simultaniously.
const MaxPossibleNumberOfCounters = 10000

// Writer creates a mmap file and writes statics and counters into it.
type Writer struct {
	filename   string
	idSequence int64
	closed     int32
	buffer     *offheap.Buffer
	encoder    *layout.Encoder
	values     *offheap.Buffer
}

// NewWriterForFile creates new instance of the Writer.
// filename specifies a path to the mmap file.
// statics contains all static values to be published.
// maxNumbersOfCounters defines how many counters are going to be created in this file maximum.
// If the file already exists, the function returns an error.
func NewWriterForFile(filename string, statics map[string]string, maxNumbersOfCounters int) (w *Writer, err error) {
	if maxNumbersOfCounters < 0 || maxNumbersOfCounters > MaxPossibleNumberOfCounters {
		return nil, fmt.Errorf("Incorrect max numbers of counters: %d", maxNumbersOfCounters)
	}

	staticsLength := layout.StaticsLength(statics)
	metadataLength := layout.MetadataLength(maxNumbersOfCounters)
	valuesLength := layout.ValuesLength(maxNumbersOfCounters)

	countersFileSize := layout.Align(
		layout.HeaderLength()+
			staticsLength+
			metadataLength+
			valuesLength,
		os.Getpagesize())

	buf, err := mmap.MapNewFile(filename, countersFileSize)
	if err != nil {
		return nil, err
	}

	encoder := layout.NewEncoder(buf,
		staticsLength,
		metadataLength,
		valuesLength)

	encoder.SetPid(int64(os.Getpid()))
	encoder.SetStartTime(time.Now().UnixNano() / int64(time.Millisecond))
	encoder.SetStatics(statics)

	encoder.SetVersion(layout.CountersVersion)

	return &Writer{
		filename:   filename,
		idSequence: -1,
		closed:     0,
		buffer:     buf,
		encoder:    encoder,
		values:     encoder.Layout.CountersValues,
	}, nil
}

// NewWriterForName creates new instance of the Writer with the given file name.
// name specifies a name of the counter's file. The file is being created in the default directory.
// statics contains all static values to be published.
// maxNumbersOfCounters defines how many counters are going to be created in this file maximum.
// If the file already exists, the function returns an error.
func NewWriterForName(name string, statics map[string]string, maxNumbersOfCounters int) (w *Writer, err error) {
	return NewWriterForFile(path.Join(GetMCountersDirectoryPath(), name), statics, maxNumbersOfCounters)
}

// Filename returns the path to the counters' file.
func (w *Writer) Filename() (filename string) {
	return w.filename
}

// Buffer returns offheap buffer to access the counters' file.
func (w *Writer) Buffer() (buf *offheap.Buffer) {
	return w.buffer
}

// AddCounter creates and returns new counter with the label specified.
func (w *Writer) AddCounter(label string) (c *Counter, err error) {
	return w.AddCounterWithInitialValue(label, 0)
}

// AddCounterWithInitialValue creates and returns new counter with the label and initial value specified.
func (w *Writer) AddCounterWithInitialValue(label string, initialValue int64) (c *Counter, err error) {
	id := atomic.AddInt64(&w.idSequence, 1)

	valueOffset, err := w.encoder.AddCounter(id, initialValue, label)
	if err != nil {
		return nil, err
	}

	return &Counter{
		owner:       w,
		id:          id,
		label:       label,
		valueOffset: valueOffset,
		closed:      0,
	}, nil
}

// IsClosed returns true if the writer was closed.
func (w *Writer) IsClosed() bool {
	return atomic.LoadInt32(&w.closed) != 0
}

// Close closes the writer and unmaps previously mapped counters' file.
func (w *Writer) Close() (err error) {
	if !atomic.CompareAndSwapInt32(&w.closed, 0, 1) {
		return
	}
	return mmap.Unmap(w.buffer)
}

// Counter presents. Note, that the counter cannot be used after the writer is closed,
// since this leads to segmentation fault.
type Counter struct {
	owner       *Writer
	id          int64
	label       string
	valueOffset uintptr
	closed      int32
}

// ID returns ID of the counter. ID is unique for the process.
func (c *Counter) ID() int64 {
	return c.id
}

// Label returns the label of the counter.
func (c *Counter) Label() string {
	return c.label
}

// Get returns the value of the counter with volatile semantic.
func (c *Counter) Get() int64 {
	return c.owner.values.GetInt64Volatile(c.valueOffset)
}

// GetWeak returns the value of the counter without volatile semantic.
func (c *Counter) GetWeak() int64 {
	return c.owner.values.GetInt64(c.valueOffset)
}

// GetWeak sets the value of the counter with volatile semantic.
func (c *Counter) Set(v int64) {
	c.owner.values.PutInt64Volatile(c.valueOffset, v)
}

// GetWeak sets the value of the counter without volatile semantic.
func (c *Counter) SetWeak(v int64) {
	c.owner.values.PutInt64(c.valueOffset, v)
}

// GetWeak increments the value of the counter with volatile semantic.
func (c *Counter) Increment() int64 {
	return c.owner.values.AddInt64(c.valueOffset, 1)
}

// GetWeak returns  the value of the counter and adds a delta to it with volatile semantic.
func (c *Counter) GetAndAdd(delta int64) int64 {
	return c.owner.values.AddInt64(c.valueOffset, delta) - delta
}

// IsClosed returns true if the counter was closed.
func (c *Counter) IsClosed() bool {
	return atomic.LoadInt32(&c.closed) != 0
}

// Close closes the counter and frees its memory slot.
func (c *Counter) Close() {
	if !atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		return
	}
	c.owner.encoder.FreeCounter(c.id)
}
