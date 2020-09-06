// Copyright (c) 2020 anatolygudkov. All rights reserved.
// Use of this source code is governed by MIT license
// that can be found in the LICENSE file.
package rest

import (
	"testing"
)

func TestPath(t *testing.T) {
	validate(t, "")
	validate(t, "/", "")
	validate(t, "//", "")
	validate(t, "abc", "abc")
	validate(t, "abc/", "abc")
	validate(t, "/abc", "", "abc")
	validate(t, "//abc", "", "abc")
	validate(t, "//abc/def", "", "abc", "def")
	validate(t, "//abc/def/g", "", "abc", "def", "g")
	validate(t, "//abc/10/def", "", "abc", "10", "def")
	validate(t, "\n\r/\t/  a bc / 10 / def /  ", "", "a bc", "10", "def")
}

func validate(t *testing.T, path string, expected ...string) {
	p := newPath(path)

	i := 0
	for p.next() {
		if i == len(expected) {
			t.Fatalf("Case: '%s'. Iteration: %d. No expected anymore, extracted: '%s'", path, i, p.segment())
		}
		if p.segment() != expected[i] {
			t.Fatalf("Case: '%s'. Iteration: %d. Expected segment: '%s', extracted: '%s'", path, i, expected[i], p.segment())
		}
		i++
	}

	if i != len(expected) {
		t.Fatalf("Case: '%s'. Expected segments: %d, iterated: %d", path, len(expected), i)
	}
}

func TestRestApp(t *testing.T) {
}
