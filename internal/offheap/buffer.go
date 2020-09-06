// Copyright (c) 2020 anatolygudkov. All rights reserved.
// Use of this source code is governed by MIT license
// that can be found in the LICENSE file.
package offheap

import (
	"sync/atomic"
	"unsafe"
)

// Buffer is a...
type Buffer struct {
	addr     uintptr
	capacity int
}

// NewBuffer creates
func NewBuffer(addr uintptr, capacity int) *Buffer {
	return &Buffer{
		addr:     addr,
		capacity: capacity,
	}
}

// Address returns
func (b *Buffer) Address() uintptr {
	return b.addr
}

// Capacity returns
func (b *Buffer) Capacity() int {
	return b.capacity
}

// Slice returns
func (b *Buffer) Slice(offset uintptr, capacity int) *Buffer {
	return NewBuffer(b.addr+offset, capacity)
}

// GetInt32 returns
func (b *Buffer) GetInt32(offset uintptr) int32 {
	return *(*int32)(unsafe.Pointer(b.addr + offset))
}

// GetInt32Volatile returns
func (b *Buffer) GetInt32Volatile(offset uintptr) int32 {
	return atomic.LoadInt32((*int32)(unsafe.Pointer(b.addr + offset)))
}

// PutInt32 sets
func (b *Buffer) PutInt32(offset uintptr, v int32) {
	*(*int32)(unsafe.Pointer(b.addr + offset)) = v
}

// PutInt32Volatile sets
func (b *Buffer) PutInt32Volatile(offset uintptr, v int32) {
	*(*int32)(unsafe.Pointer(b.addr + offset)) = v
}

// GetInt64 returns
func (b *Buffer) GetInt64(offset uintptr) int64 {
	return *(*int64)(unsafe.Pointer(b.addr + offset))
}

// GetInt64Volatile returns
func (b *Buffer) GetInt64Volatile(offset uintptr) int64 {
	return atomic.LoadInt64((*int64)(unsafe.Pointer(b.addr + offset)))
}

// PutInt64 sets
func (b *Buffer) PutInt64(offset uintptr, v int64) {
	*(*int64)(unsafe.Pointer(b.addr + offset)) = v
}

// PutInt64Volatile sets
func (b *Buffer) PutInt64Volatile(offset uintptr, v int64) {
	atomic.StoreInt64((*int64)(unsafe.Pointer(b.addr+offset)), v)
}

// AddInt64 sets
func (b *Buffer) AddInt64(offset uintptr, delta int64) int64 {
	return atomic.AddInt64((*int64)(unsafe.Pointer(b.addr+offset)), delta)
}

// SwapInt64 sets
func (b *Buffer) SwapInt64(offset uintptr, new int64) (old int64) {
	return atomic.SwapInt64((*int64)(unsafe.Pointer(b.addr+offset)), old)
}

// CompareAndSwapInt64 sets
func (b *Buffer) CompareAndSwapInt64(offset uintptr, old, new int64) bool {
	return atomic.CompareAndSwapInt64((*int64)(unsafe.Pointer(b.addr+offset)), old, new)
}

// PutString puts
func (b *Buffer) PutString(offset uintptr, s string) {
	b.PutBytes(offset, []byte(s))
}

// PutBytes sets
func (b *Buffer) PutBytes(offset uintptr, bs []byte) {
	b.PutSomeBytes(offset, bs, 0, len(bs))
}

// PutSomeBytes sets
func (b *Buffer) PutSomeBytes(offset uintptr, bs []byte, start, len int) {
	var s = struct {
		addr uintptr
		len  int
		cap  int
	}{b.addr + offset, len, len}

	dest := *(*[]byte)(unsafe.Pointer(&s))

	copy(dest, bs[start:start+len])
}

// GetBytes gets
func (b *Buffer) GetBytes(offset uintptr, length int) (bs []byte) {
	var s = struct {
		addr uintptr
		len  int
		cap  int
	}{b.addr + offset, length, length}

	src := *(*[]byte)(unsafe.Pointer(&s))

	bs = make([]byte, length)

	copy(bs, src)

	return bs
}

// GetString gets
func (b *Buffer) GetString(offset uintptr, length int) string {
	return string(b.GetBytes(offset, length))
}
