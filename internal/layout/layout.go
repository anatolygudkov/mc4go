// Copyright (c) 2020 anatolygudkov. All rights reserved.
// Use of this source code is governed by MIT license
// that can be found in the LICENSE file.
package layout

import (
	"github.com/anatolygudkov/mc4go/internal/offheap"
)

// CountersVersion presents
const CountersVersion = 1

const sizeOfInt32 = 4
const sizeOfInt64 = 8
const sizeOfCacheLine = 64

/**
 * Layout of the counters.
 *
 * Header
 *
 *   0                   1                   2                   3
 *   0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
 *  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
 *  |                      Counters version                         |
 *  +---------------------------------------------------------------+
 *  |                       Statics length                          |
 *  +---------------------------------------------------------------+
 *  |                       Metadata length                         |
 *  +---------------------------------------------------------------+
 *  |                        Values length                          |
 *  +---------------------------------------------------------------+
 *  |                             PID                               |
 *  |                                                               |
 *  +---------------------------------------------------------------+
 *  |                      Start time millis                        |
 *  |                                                               |
 *  +---------------------------------------------------------------+
 *  |                     96 bytes of padding                      ...
 * ...                                                              |
 *  +---------------------------------------------------------------+
 *
 *
 * Statics
 *
 *   0                   1                   2                   3
 *   0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
 *  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
 *  |                      Number of statics                        |
 *  +---------------------------------------------------------------+
 *  |                       Static[0]'s label length                |
 *  +---------------------------------------------------------------+
 *  |                      Static[0]'s value length                 |
 *  +---------------------------------------------------------------+
 *  |                       Static[0]'s label                      ...
 * ...                                                              |
 *  +---------------------------------------------------------------+
 *  |                       Static[0]'s value                      ...
 * ...                                                              |
 *  +---------------------------------------------------------------+
 *  |   Some bytes of padding to have Static[1]'s label length     ...
 * ...                     aligned on 4 bytes                       |
 *  +---------------------------------------------------------------+
 *  |               Repeats for Static[1]-Static[N]                ...
 *  |                                                               |
 * ...                                                              |
 *  +---------------------------------------------------------------+
 *  |                   Some bytes of padding to have              ...
 * ...               this section aligned on 128 bytes              |
 *  +---------------------------------------------------------------+
 *
 *
 * Metadata
 *
 *   0                   1                   2                   3
 *   0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
 *  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
 *  |                Counter[0]'s ID << 8 | Status                  |
 *  |                                                               |
 *  +---------------------------------------------------------------+
 *  |                     120 bytes of padding                     ...
 * ...                                                              |
 *  +---------------------------------------------------------------+
 *  |                  Counters[0]'s label length                   |
 *  +---------------------------------------------------------------+
 *  |            380 bytes of the Counters[0]'s label              ...
 * ...                                                              |
 *  +---------------------------------------------------------------+
 *  |              Repeats for Counter[1]-Counter[N]               ...
 *  |                                                               |
 * ...                                                              |
 *  +---------------------------------------------------------------+
 *
 *
 * Values
 *
 *   0                   1                   2                   3
 *   0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
 *  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
 *  |                       Counter[0]'s value                      |
 *  |                                                               |
 *  +---------------------------------------------------------------+
 *  |                     120 bytes of padding                     ...
 * ...                                                              |
 *  +---------------------------------------------------------------+
 *  |              Repeats for Counter[1]-Counter[N]               ...
 *  |                                                               |
 * ...                                                              |
 *  +---------------------------------------------------------------+
 */

type Layout struct {
	Header           *offheap.Buffer
	Statics          *offheap.Buffer
	CountersMetadata *offheap.Buffer
	CountersValues   *offheap.Buffer
}

const (
	headerCountersVersionOffset = 0
	headerStaticsLengthOffset   = headerCountersVersionOffset + sizeOfInt32
	headerMetadataLengthOffset  = headerStaticsLengthOffset + sizeOfInt32
	headerValuesLengthOffset    = headerMetadataLengthOffset + sizeOfInt32
	headerPidOffsert            = headerValuesLengthOffset + sizeOfInt32
	headerStartTimeOffsert      = headerPidOffsert + sizeOfInt64
)

func HeaderLength() int {
	return Align(headerStartTimeOffsert+sizeOfInt64, sizeOfCacheLine*2)
}

const (
	staticsNumberOfStaticsOffset = 0
	staticsRecordsOffset         = staticsNumberOfStaticsOffset + sizeOfInt32
)

const (
	staticsLabelLengthOffset = 0
	staticsValueLengthOffset = staticsLabelLengthOffset + sizeOfInt32
	staticsLabelOffset       = staticsValueLengthOffset + sizeOfInt32
)

func staticsRecordLength(labelLength int, valueLength int) int {
	return Align(staticsLabelOffset+labelLength+valueLength, sizeOfInt32) // should be aligned to have next labelLength and valueLength
	// integers aligned
}

const (
	metadataLabelMaxLength        = sizeOfCacheLine*6 - sizeOfInt32 // max length of the label's text without its length prefix
	metadataCounterIDStatusOffset = 0
	metadataLabelLengthOffset     = sizeOfCacheLine * 2
	metadataLabelOffset           = metadataLabelLengthOffset + sizeOfInt32
	metadataRecordLength          = metadataLabelOffset + metadataLabelMaxLength
)

const valuesCounterLength = sizeOfCacheLine * 2

const (
	counterStatusNotUsed              uint8 = 0
	counterStatusAllocationInProgress uint8 = 1
	counterStatusAllocated            uint8 = 2
	counterStatusFreed                uint8 = 3
)

func makeIDStatus(id int64, status uint8) int64 {
	return int64(uint64(id)<<8 | uint64(status))
}

func extractStatus(idStatus int64) uint8 {
	return uint8(idStatus & 0xff)
}

func extractID(idStatus int64) int64 {
	return int64(uint64(idStatus) >> 8)
}

// Align rounds v up to alignment multiple of alignment. alignment must be a power of 2.
func Align(v int, alignment int) int {
	return (v + alignment - 1) &^ (alignment - 1)
}
