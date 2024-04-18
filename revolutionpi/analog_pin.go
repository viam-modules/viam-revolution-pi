//go:build linux

// Package revolutionpi implements the Revolution Pi board GPIO pins.
package revolutionpi

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
)

type analogPin struct {
	Name         string // Variable name
	Address      uint16 // Address of the byte in the process image
	Length       uint16 // length of the variable in bits. Possible values are 1, 8, 16 and 32
	ControlChip  *gpioChip
	outputOffset uint16
	inputOffset  uint16
}

func (pin *analogPin) Read(ctx context.Context, extra map[string]interface{}) (int, error) {
	if !pin.isAnalogInput() {
		return 0, fmt.Errorf("cannot ReadAnalog, pin %s is not an analog input pin", pin.Name)
	}
	pin.ControlChip.logger.Debugf("Reading from %v, length: %v byte(s)", pin.Address, pin.Length/8)
	b := make([]byte, pin.Length/8)
	n, err := pin.ControlChip.fileHandle.ReadAt(b, int64(pin.Address))
	pin.ControlChip.logger.Debugf("Read %#v bytes", b)
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

func (b *revolutionPiBoard) WriteAnalog(ctx context.Context, pin string, value int32, extra map[string]interface{}) error {
	analogPin, err := b.controlChip.GetAnalogPin(pin)
	if err != nil {
		return err
	}
	b.logger.Infof("Analog: %#v", analogPin)
	if !analogPin.isAnalogOutput() {
		return fmt.Errorf("cannot WriteAnalog, pin %s is not an analog output pin", pin)
	}
	buf := new(bytes.Buffer)
	err = binary.Write(buf, binary.LittleEndian, value)
	if err != nil {
		return err
	}
	return analogPin.ControlChip.writeValue(int64(analogPin.Address), buf.Bytes())
}

// pins at 0 or 2 + outputOffset.
func (pin *analogPin) isAnalogOutput() bool {
	return pin.Address == pin.outputOffset || pin.Address == pin.outputOffset+2
}

// pins at 0-7 + inputOffset.
func (pin *analogPin) isAnalogInput() bool {
	return pin.Address >= pin.inputOffset && pin.Address < pin.inputOffset+8
}
