// Copyright (c) 2020 anatolygudkov. All rights reserved.
// Use of this source code is governed by MIT license
// that can be found in the LICENSE file.
package mc4go

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path"
	"runtime"

	"github.com/anatolygudkov/mc4go/internal/layout"
	"github.com/anatolygudkov/mc4go/internal/mmap"
	"github.com/anatolygudkov/mc4go/internal/offheap"
)

// GetMCountersDirectoryPath returns
func GetMCountersDirectoryPath() (p string) {
	p = os.Getenv("mcounters.dir")
	if p != "" {
		return
	}

	baseDir := ""
	if runtime.GOOS == `linux` {
		shm := "/dev/shm"
		_, err := os.Stat(shm)
		if err == nil {
			baseDir = shm
		}
	}
	if baseDir == "" {
		baseDir = os.TempDir()
	}

	username := "default"
	u, err := user.Current()
	if err == nil {
		if u.Username != "" {
			username = u.Username
		}
	}

	p = path.Join(baseDir, "mcounters-"+username)
	return
}

// Reader reads
type Reader struct {
	buffer  *offheap.Buffer
	decoder *layout.Decoder
}

// NewReader creates
func NewReader(buf *offheap.Buffer) (r *Reader, err error) {
	decoder := layout.NewDecoder(buf)

	version := decoder.Version()
	if version == 0 {
		return nil, errors.New("counters haven't been initialized yet")
	}
	if version != layout.CountersVersion {
		return nil, fmt.Errorf("unexpected version of the counters file: %d", version)
	}

	return &Reader{
		buffer:  buf,
		decoder: decoder,
	}, nil
}

// NewReaderForFile creates
func NewReaderForFile(filename string) (r *Reader, err error) {
	buf, err := mmap.MapExistingFileReadOnly(filename)
	if err != nil {
		return nil, err
	}
	return NewReader(buf)
}

// NewReaderForName creates
func NewReaderForName(name string) (r *Reader, err error) {
	return NewReaderForFile(path.Join(GetMCountersDirectoryPath(), name))
}

// Version returns
func (r *Reader) Version() int32 {
	return r.decoder.Version()
}

// Pid returns
func (r *Reader) Pid() int64 {
	return r.decoder.Pid()
}

// StartTime returns
func (r *Reader) StartTime() int64 {
	return r.decoder.StartTime()
}

// ForEachStatic returns
func (r *Reader) ForEachStatic(consumer func(label, value string) bool) {
	r.decoder.ForEachStatic(consumer)
}

// GetStaticValue returns
func (r *Reader) GetStaticValue(label string) (v string, err error) {
	return r.decoder.GetStaticValue(label)
}

// ForEachCounter returns
func (r *Reader) ForEachCounter(consumer func(id, value int64, label string) bool) {
	r.decoder.ForEachCounter(consumer)
}

// GetCounterValue returns
func (r *Reader) GetCounterValue(counterID int64) (value int64, err error) {
	return r.decoder.GetCounterValue(counterID)
}

// GetCounterLabel returns
func (r *Reader) GetCounterLabel(counterID int64) (label string, err error) {
	return r.decoder.GetCounterLabel(counterID)
}

// Close returns
func (r *Reader) Close() (err error) {
	return mmap.Unmap(r.buffer)
}
