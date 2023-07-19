//go:build linux

// Package revolution_pi implements the Revolution Pi board GPIO pins.
package revolution_pi

import (
	"context"
	"encoding/binary"
	"fmt"
)

type analogPin struct {
	Name        string // Variable name
	Address     uint16 // Address of the byte in the process image
	Length      uint16 // length of the variable in bits. Possible values are 1, 8, 16 and 32
	ControlChip *gpioChip
}

func (pin *analogPin) Read(ctx context.Context, extra map[string]interface{}) (int, error) {
	pin.ControlChip.logger.Infof("Reading from %v, length: %v byte(s)", pin.Address, pin.Length/8)
	b := make([]byte, pin.Length/8)
	n, err := pin.ControlChip.fileHandle.ReadAt(b, int64(pin.Address))
	pin.ControlChip.logger.Infof("Read %#v bytes", b)
	if n != 2 {
		return 0, fmt.Errorf("expected 2 bytes, got %#v", b)
	}
	if err != nil {
		return 0, err
	}
	val := binary.LittleEndian.Uint16(b)
	return int(val), nil
}

func (pin *analogPin) Close(ctx context.Context) error {
	// There is nothing to close with respect to individual analog _reader_ pins
	return nil
}
