// Copyright (c) 2020 anatolygudkov. All rights reserved.
// Use of this source code is governed by MIT license
// that can be found in the LICENSE file.
package layout

import (
	"errors"
	"fmt"
	"sort"

	"github.com/anatolygudkov/mc4go/internal/offheap"
)

// StaticsLength returns
func StaticsLength(statics map[string]string) (l int) {
	l = staticsRecordsOffset // some space for number of statics

	if statics == nil || len(statics) == 0 {
		return l
	}

	var labels []string
	for label := range statics {
		labels = append(labels, label)
	}
	sort.Strings(labels)
	for _, label := range labels {
		value := statics[label]

		labelBytes := []byte(label)
		valueBytes := []byte(value)

		l += staticsRecordLength(len(labelBytes), len(valueBytes))
	}

	l = Align(l, sizeOfCacheLine*2)

	return l
}

// MetadataLength returns
func MetadataLength(numberOfCounters int) int {
	return numberOfCounters * metadataRecordLength
}

// ValuesLength returns
func ValuesLength(numberOfCounters int) int {
	return numberOfCounters * valuesCounterLength
}

// Encoder struct
type Encoder struct {
	Layout Layout
}

// NewEncoder creates
func NewEncoder(buf *offheap.Buffer, staticsLength, metadataLength, valuesLength int) *Encoder {
	return NewEncoderWithBuffers(buf.Slice(0, HeaderLength()),
		buf.Slice(uintptr(HeaderLength()), staticsLength),
		buf.Slice(uintptr(HeaderLength()+staticsLength), metadataLength),
		buf.Slice(uintptr(HeaderLength()+staticsLength+metadataLength), valuesLength),
	)
}

// NewEncoderWithBuffers creates
func NewEncoderWithBuffers(header, statics, countersMetadata, countersValues *offheap.Buffer) *Encoder {
	e := Encoder{
		Layout: Layout{
			Header:           header,
			Statics:          statics,
			CountersMetadata: countersMetadata,
			CountersValues:   countersValues,
		},
	}

	header.PutInt32(headerStaticsLengthOffset, int32(statics.Capacity()))
	header.PutInt32(headerMetadataLengthOffset, int32(countersMetadata.Capacity()))
	header.PutInt32(headerValuesLengthOffset, int32(countersValues.Capacity()))
	// These writes will be finished by a membar of write of VERSION (SetVersion call)
	// at the end of the header's preparation.

	return &e
}

// SetVersion sets
func (e *Encoder) SetVersion(v int32) {
	e.Layout.Header.PutInt32Volatile(headerCountersVersionOffset, v)
}

// SetPid sets
func (e *Encoder) SetPid(p int64) {
	e.Layout.Header.PutInt64Volatile(headerPidOffsert, p)
}

// SetStartTime sets
func (e *Encoder) SetStartTime(t int64) {
	e.Layout.Header.PutInt64Volatile(headerStartTimeOffsert, t)
}

// SetStatics sets
func (e *Encoder) SetStatics(statics map[string]string) (err error) {
	statx := e.Layout.Statics

	offset := 0

	if statics == nil || len(statics) == 0 {
		statx.PutInt32Volatile(uintptr(offset), 0)
		return
	}

	if offset+staticsRecordsOffset > statx.Capacity() {
		return fmt.Errorf("statics buffer is too small %d", statx.Capacity())
	}

	var labels []string
	for label := range statics {
		labels = append(labels, label)
	}
	sort.Strings(labels)

	statx.PutInt32Volatile(uintptr(offset), int32(len(labels)))

	offset = staticsRecordsOffset

	for _, label := range labels {
		value := statics[label]

		labelBytes := []byte(label)
		valueBytes := []byte(value)

		recordLength := staticsRecordLength(len(labelBytes), len(valueBytes))

		if offset+recordLength > statx.Capacity() {
			return fmt.Errorf("properties don't feet to the statics's buffer size %d", statx.Capacity())
		}

		statx.PutBytes(uintptr(offset+staticsLabelOffset), labelBytes)
		statx.PutBytes(uintptr(offset+staticsLabelOffset+len(labelBytes)), valueBytes)

		statx.PutInt32(uintptr(offset+staticsLabelLengthOffset), int32(len(labelBytes)))
		statx.PutInt32Volatile(uintptr(offset+staticsValueLengthOffset), int32(len(valueBytes)))

		offset += recordLength
	}

	return nil
}

// AddCounter adds
func (e *Encoder) AddCounter(id, initialValue int64, label string) (valueOffset uintptr, err error) {
	metadata := e.Layout.CountersMetadata
	values := e.Layout.CountersValues

	metadataOffset := 0
	valueOffset = 0

	for metadataOffset < metadata.Capacity() {
		idStatusOffset := metadataOffset + metadataCounterIDStatusOffset

		idStatus := metadata.GetInt64Volatile(uintptr(idStatusOffset))

		status := extractStatus(idStatus)

		switch status {
		case counterStatusNotUsed, counterStatusFreed:
			inProgressIDStatus := makeIDStatus(id, counterStatusAllocationInProgress)

			if metadata.CompareAndSwapInt64(uintptr(idStatusOffset), idStatus, inProgressIDStatus) {

				labelBytes := []byte(label)

				labelLength := len(labelBytes)
				if metadataLabelMaxLength < labelLength {
					labelLength = metadataLabelMaxLength
				}

				metadata.PutInt32(uintptr(metadataOffset+metadataLabelLengthOffset), int32(labelLength))
				metadata.PutSomeBytes(uintptr(metadataOffset+metadataLabelOffset), labelBytes, 0, labelLength)

				values.PutInt64(uintptr(valueOffset), initialValue)

				allocatedIDStatus := makeIDStatus(id, counterStatusAllocated)

				metadata.PutInt64Volatile(uintptr(idStatusOffset), allocatedIDStatus)

				return valueOffset, nil
			}
			continue

		default:
		}

		metadataOffset += metadataRecordLength
		valueOffset += valuesCounterLength
	}

	return 0, errors.New("there is no free space to add new counter")
}

// FreeCounter frees the memory slot occupied by the counter.
func (e *Encoder) FreeCounter(id int64) (success bool) {
	metadata := e.Layout.CountersMetadata

	metadataOffset := 0

	for metadataOffset < metadata.Capacity() {
		idStatusOffset := metadataOffset + metadataCounterIDStatusOffset

		idStatus := metadata.GetInt64Volatile(uintptr(idStatusOffset))

		currentID := extractID(idStatus)

		if currentID == id {
			status := extractStatus(idStatus)

			switch status {
			case counterStatusAllocated:
				newIDStatus := makeIDStatus(id, counterStatusFreed)
				metadata.CompareAndSwapInt64(uintptr(idStatusOffset), idStatus, newIDStatus) // we don't care
				// about result of CAS, since the counter may be freed by another thread already
				// (a race condition) and this is good for us anyway
				return true
			default:
				return false
			}
		}

		metadataOffset += metadataRecordLength
	}
	return false
}
