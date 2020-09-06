// Copyright (c) 2020 anatolygudkov. All rights reserved.
// Use of this source code is governed by MIT license
// that can be found in the LICENSE file.
package offheap

import (
	"testing"
	"unsafe"
)

func newBuffer() *Buffer {
	bytes := make([]byte, 1000)
	return NewBuffer(uintptr(unsafe.Pointer(&bytes)), cap(bytes))
}

func TestBuffer(t *testing.T) {
	buffer := newBuffer()

	buffer.PutSomeBytes(2, []byte("atestb"), 2, 4)

	s := string(buffer.GetString(2, 4))

	sexp := "est"

	if s != sexp {
		t.Fatalf("Bytes not matched. Expected: %s, got %s", sexp, s)
	}
}
