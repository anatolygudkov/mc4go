// Copyright (c) 2020 anatolygudkov. All rights reserved.
// Use of this source code is governed by MIT license
// that can be found in the LICENSE file.
// +build !windows,!plan9,!solaris,!aix
package mmap

import (
	"os"
	"syscall"
	"unsafe"
)

// mmap maps
func mmap(f *os.File, readOnly bool) (addr uintptr, size int, err error) {
	fi, err := f.Stat()
	if err != nil {
		return 0, 0, err
	}

	prot := syscall.PROT_READ
	if !readOnly {
		prot |= syscall.PROT_WRITE
	}

	size = int(fi.Size())

	b, err := syscall.Mmap(int(f.Fd()), 0, size, prot, syscall.MAP_SHARED)
	if err != nil {
		return 0, 0, err
	}

	return uintptr(unsafe.Pointer(&b[0])), size, nil
}

// munmap maps
func munmap(addr uintptr, size int) (err error) {
	var s = struct {
		addr uintptr
		len  int
		cap  int
	}{addr, size, size}
	return syscall.Munmap(*(*[]byte)(unsafe.Pointer(&s)))
}
