//go:build linux

// Package revolutionpi implements the Revolution Pi board GPIO pins.
package revolutionpi

import (
	"encoding/binary"
	"fmt"
)

func str32(chars [32]byte) string {
	i := 0
	var c byte
	for i, c = range chars {
		if c == 0 {
			break
		}
	}
	return string(chars[:i])
}

func char32(str string) (chars [32]byte) {
	copy(chars[:31], str)
	return
}

func readFromBuffer(buf []byte, size int) (interface{}, error) {
	switch size {
	case 1:
		return buf[0], nil
	case 2:
		return binary.LittleEndian.Uint16(buf), nil
	case 4:
		return binary.LittleEndian.Uint32(buf), nil
	default:
		return nil, fmt.Errorf("unexpected byte size, got %v bytes", size)
	}
}
