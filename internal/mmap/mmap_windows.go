// Copyright (c) 2020 anatolygudkov. All rights reserved.
// Use of this source code is governed by MIT license
// that can be found in the LICENSE file.
// Not tested yet!
package mmap

import (
	"os"
	"syscall"
)

// mmap maps
func mmap(f *os.File, readOnly bool) (addr uintptr, size int, err error) {
	fi, err := f.Stat()
	if err != nil {
		return 0, 0, err
	}

	prot := uint32(syscall.PAGE_READONLY)
	access := uint32(syscall.FILE_MAP_READ)
	if !readOnly {
		prot = uint32(syscall.PAGE_READWRITE)
		access = uint32(syscall.FILE_MAP_WRITE)
	}

	h, errno := syscall.CreateFileMapping(syscall.Handle(f.Fd()), nil, prot, 0, 0, nil)
	if handle == 0 {
		return 0, 0, os.NewSyscallError("CreateFileMapping", errno)
	}

	size = fi.Size()

	addr, errno = syscall.MapViewOfFile(h, access, 0, 0, size)
	if addr == 0 {
		return 0, 0, os.NewSyscallError("MapViewOfFile", errno)
	}

	if err := syscall.CloseHandle(syscall.Handle(h)); err != nil {
		return 0, 0, os.NewSyscallError("CloseHandle", err)
	}

	return addr, size, nil
}

// munmap unmaps
func munmap(addr uintptr) (err error) {
	if err := syscall.UnmapViewOfFile(addr); err != nil {
		return os.NewSyscallError("UnmapViewOfFile", err)
	}
	return nil
}
