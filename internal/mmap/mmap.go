// Copyright (c) 2020 anatolygudkov. All rights reserved.
// Use of this source code is governed by MIT license
// that can be found in the LICENSE file.
package mmap

import (
	"os"
	"path"

	"github.com/anatolygudkov/mc4go/internal/offheap"
)

// MapNewFile My func
func MapNewFile(filename string, size int) (buf *offheap.Buffer, err error) {
	pageSize := os.Getpagesize()

	alignedSize := align(size, pageSize)

	dir := path.Dir(filename)
	os.MkdirAll(dir, os.ModePerm)

	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	err = os.Truncate(f.Name(), int64(alignedSize))
	if err != nil {
		return nil, err
	}

	addr, _, err := mmap(f, false)
	if err != nil {
		return nil, err
	}

	buf = offheap.NewBuffer(addr, alignedSize)

	// Now pre-touch all the pages.
	position := 0
	for position < alignedSize {
		buf.PutInt64(uintptr(position), 0)
		position += pageSize
	}

	return buf, nil
}

// MapExistingFileReadOnly maps
func MapExistingFileReadOnly(filename string) (buf *offheap.Buffer, err error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	addr, size, err := mmap(file, true)
	if err != nil {
		return nil, err
	}

	return offheap.NewBuffer(addr, size), nil
}

// Unmap unpams
func Unmap(buf *offheap.Buffer) (err error) {
	return munmap(buf.Address(), buf.Capacity())
}

// align rounds v up to alignment multiple of alignment. alignment must be a power of 2.
func align(v int, alignment int) int {
	return (v + alignment - 1) &^ (alignment - 1)
}
