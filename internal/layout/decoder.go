// Copyright (c) 2020 anatolygudkov. All rights reserved.
// Use of this source code is governed by MIT license
// that can be found in the LICENSE file.
package layout

import (
	"bytes"
	"fmt"

	"github.com/anatolygudkov/mc4go/internal/offheap"
)

// Decoder decodes
type Decoder struct {
	Layout Layout
}

// NewDecoder creates
func NewDecoder(buf *offheap.Buffer) *Decoder {
	header := buf.Slice(0, HeaderLength())

	staticsLength := int(header.GetInt32Volatile(headerStaticsLengthOffset))
	metadataLength := int(header.GetInt32(headerMetadataLengthOffset))
	valuesLength := int(header.GetInt32(headerValuesLengthOffset))

	return &Decoder{
		Layout: Layout{
			Header:           header,
			Statics:          buf.Slice(uintptr(HeaderLength()), staticsLength),
			CountersMetadata: buf.Slice(uintptr(HeaderLength()+int(staticsLength)), metadataLength),
			CountersValues:   buf.Slice(uintptr(HeaderLength()+int(staticsLength+metadataLength)), valuesLength),
		},
	}
}

// NewDecoderWithBuffers creates
func NewDecoderWithBuffers(header, statics, countersMetadata, countersValues *offheap.Buffer) *Decoder {
	return &Decoder{
		Layout: Layout{
			Header:           header,
			Statics:          statics,
			CountersMetadata: countersMetadata,
			CountersValues:   countersValues,
		},
	}
}

// Version returns
func (d *Decoder) Version() int32 {
	return d.Layout.Header.GetInt32Volatile(headerCountersVersionOffset)
}

// Pid returns
func (d *Decoder) Pid() int64 {
	return d.Layout.Header.GetInt64Volatile(headerPidOffsert)
}

// StartTime returns
func (d *Decoder) StartTime() int64 {
	return d.Layout.Header.GetInt64Volatile(headerStartTimeOffsert)
}

// ForEachStatic returns
func (d *Decoder) ForEachStatic(consumer func(label, value string) bool) {
	statics := d.Layout.Statics

	offset := staticsNumberOfStaticsOffset

	numOfStatics := int(statics.GetInt32Volatile(uintptr(offset)))

	offset = staticsRecordsOffset

	for i := 0; i < numOfStatics; i++ {
		labelLen := int(statics.GetInt32(uintptr(offset + staticsLabelLengthOffset)))
		valueLen := int(statics.GetInt32(uintptr(offset + staticsValueLengthOffset)))

		label := statics.GetString(uintptr(offset+staticsLabelOffset), labelLen)
		value := statics.GetString(uintptr(offset+staticsLabelOffset+labelLen), valueLen)

		if !consumer(label, value) {
			return
		}

		recordLen := staticsRecordLength(int(labelLen), int(valueLen))

		offset += int(recordLen)
	}
}

// GetStaticValue returns
func (d *Decoder) GetStaticValue(label string) (v string, err error) {
	offset := staticsNumberOfStaticsOffset

	statics := d.Layout.Statics

	numOfStatics := int(statics.GetInt32Volatile(uintptr(offset)))

	offset = staticsRecordsOffset

	staticLabelBytes := []byte(label)

	for i := 0; i < numOfStatics; i++ {
		labelLength := int(statics.GetInt32(uintptr(offset + staticsLabelLengthOffset)))
		valueLength := int(statics.GetInt32(uintptr(offset + staticsValueLengthOffset)))

		labelBytes := statics.GetBytes(uintptr(offset+staticsLabelOffset), labelLength)

		if bytes.Compare(staticLabelBytes, labelBytes) == 0 {
			valueBytes := statics.GetBytes(uintptr(offset+staticsLabelOffset+labelLength), valueLength)
			return string(valueBytes), nil
		}

		recordLength := staticsRecordLength(labelLength, valueLength)

		offset += recordLength
	}

	return "", fmt.Errorf("label %s isn't found", label)
}

// ForEachCounter iterates
func (d *Decoder) ForEachCounter(consumer func(id, value int64, label string) bool) {
	metadata := d.Layout.CountersMetadata
	values := d.Layout.CountersValues

	metadataOffset := 0
	valueOffset := 0

Stop:
	for metadataOffset < metadata.Capacity() {
		idStatusOffset := metadataOffset + metadataCounterIDStatusOffset

		idStatus := metadata.GetInt64Volatile(uintptr(idStatusOffset))

		switch status := extractStatus(idStatus); status {
		case counterStatusNotUsed:
			break Stop

		case counterStatusAllocated:
			id := extractID(idStatus)

			labelLength := int(metadata.GetInt32(uintptr(metadataOffset) + metadataLabelLengthOffset))

			label := metadata.GetString(uintptr(metadataOffset+metadataLabelOffset), labelLength)

			value := values.GetInt64(uintptr(valueOffset))

			// Make sure the counter's status wasn't changed yet to guarantee
			// the value just read belongs to this counter.
			if metadata.GetInt64Volatile(uintptr(idStatusOffset)) == idStatus {
				if !consumer(id, value, label) {
					return
				}
			}

		default:
		}

		metadataOffset += metadataRecordLength
		valueOffset += valuesCounterLength
	}
}

// GetCounterValue returns
func (d *Decoder) GetCounterValue(counterID int64) (value int64, err error) {
	metadata := d.Layout.CountersMetadata
	values := d.Layout.CountersValues

	metadataOffset := 0
	valueOffset := 0

	for metadataOffset < metadata.Capacity() {
		idStatusOffset := metadataOffset + metadataCounterIDStatusOffset

		idStatus := metadata.GetInt64Volatile(uintptr(idStatusOffset))

		status := extractStatus(idStatus)

		if status == counterStatusNotUsed {
			break
		}

		id := extractID(idStatus)

		if counterID == id {
			switch status {
			case counterStatusAllocated:
				value = values.GetInt64(uintptr(valueOffset))

				// Make sure the counter's status wasn't changed yet to guarantee
				// the value just read belongs to this counter.
				if metadata.GetInt64Volatile(uintptr(idStatusOffset)) == idStatus {
					return value, nil
				}
				continue

			default:
				return 0, fmt.Errorf("counter %d isn't allocated", counterID)
			}
		}

		metadataOffset += metadataRecordLength
		valueOffset += valuesCounterLength
	}

	return 0, fmt.Errorf("counter %d not found", counterID)
}

// GetCounterLabel returns
func (d *Decoder) GetCounterLabel(counterID int64) (label string, err error) {
	metadata := d.Layout.CountersMetadata

	metadataOffset := 0

	for metadataOffset < metadata.Capacity() {
		idStatusOffset := metadataOffset + metadataCounterIDStatusOffset

		idStatus := metadata.GetInt64Volatile(uintptr(idStatusOffset))

		status := extractStatus(idStatus)

		if status == counterStatusNotUsed {
			break
		}

		id := extractID(idStatus)

		if counterID == id {
			switch status {
			case counterStatusAllocated:
				labelLength := int(metadata.GetInt32(uintptr(metadataOffset) + metadataLabelLengthOffset))

				labelBytes := metadata.GetBytes(uintptr(metadataOffset+metadataLabelOffset), labelLength)

				// Make sure the counter's status wasn't changed yet to guarantee
				// the value just read belongs to this counter.
				if metadata.GetInt64Volatile(uintptr(idStatusOffset)) == idStatus {
					return string(labelBytes), nil
				}
				continue

			default:
				return "", fmt.Errorf("counter %d isn't allocated", counterID)
			}
		}

		metadataOffset += metadataRecordLength
	}

	return "", fmt.Errorf("counter %d not found", counterID)
}
