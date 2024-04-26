//go:build linux

// Package genericlinux is for Linux boards, and this particular file is for digital interrupt pins
// using the ioctl interface, indirectly by way of mkch's gpio package.
package revolutionpi

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"go.viam.com/rdk/components/board"
	"go.viam.com/rdk/grpc"
)

const (
	inputModeOffset = 88
)

type digitalInterrupt struct {
	PinName          string // Variable name
	Address          uint16 // Address of the byte in the process image
	Length           uint16 // length of the variable in bits. Possible values are 1, 8, 16 and 32
	BitPosition      uint8  // 0-7 bit position, >= 8 whole byte, only used if the digital input pin is given
	ControlChip      *gpioChip
	outputOffset     uint16
	inputOffset      uint16
	enabled          bool
	interruptAddress uint16
}

func (di *digitalInterrupt) initialize() error {
	var addressInputMode uint16

	// read from the input mode byte to determine if the pin is configured for counter/interrupt mode
	// determine which address to check for the input mode based on which pin was given in the request
	if di.isInputCounter() {
		addressInputMode = (di.Address - di.inputOffset - inputWordToCounterOffset) >> 2

		// record the address for the interrupt
		di.interruptAddress = di.Address
	} else if di.isDigitalInput() {
		// depending on whether the request came from the first or second set of digital input pins, add 0 or 8
		firstOrSecondHalf := (di.Address - di.inputOffset) << 3
		addressInputMode = firstOrSecondHalf + uint16(di.BitPosition)
		di.interruptAddress = di.inputOffset + inputWordToCounterOffset + addressInputMode*4
		di.ControlChip.logger.Info("half: ", firstOrSecondHalf)
		di.ControlChip.logger.Info("bit : ", di.BitPosition)
		di.ControlChip.logger.Info("pin number: ", addressInputMode)
		di.ControlChip.logger.Info("address counter: ", di.interruptAddress)
	} else {
		return errors.New("pin is not a digital input pin")
	}
	b := make([]byte, 1)
	// read from the input mode addresses to see if the pin is configured for interrupts
	n, err := di.ControlChip.fileHandle.ReadAt(b, int64(di.inputOffset+inputModeOffset+addressInputMode))
	if err != nil {
		return err
	}
	if n != 1 {
		return errors.New("unable to read digital input pin configuration")
	}
	di.ControlChip.logger.Debugf("Current Pin configuration: %#d", b)

	// check if the pin is configured as a counter
	if b[0] == 0 || b[0] == 3 {
		return errors.New("pin is not configured as a counter")
	}
	di.enabled = true

	return nil
}

func (di *digitalInterrupt) Value(ctx context.Context, extra map[string]interface{}) (int64, error) {
	if !di.enabled {
		return 0, fmt.Errorf("cannot get digital interrupt value, pin %s is not configured as an interrupt", di.PinName)
	}
	di.ControlChip.logger.Debugf("Reading from %d, length: 4 byte(s)", di.interruptAddress)
	b := make([]byte, 4)
	n, err := di.ControlChip.fileHandle.ReadAt(b, int64(di.interruptAddress))
	if err != nil {
		return 0, err
	}
	di.ControlChip.logger.Debugf("Read %#v bytes", b)
	if n != 4 {
		return 0, fmt.Errorf("expected 4 bytes, got %#v", b)
	}
	val := binary.LittleEndian.Uint32(b)
	return int64(val), nil
}

func (di *digitalInterrupt) Name() string {
	return di.PinName
}

func (di *digitalInterrupt) Tick(ctx context.Context, high bool, nanoseconds uint64) error {
	return grpc.UnimplementedError
}

func (di *digitalInterrupt) AddCallback(c chan board.Tick) {}

func (di *digitalInterrupt) RemoveCallback(c chan board.Tick) {}

func (di *digitalInterrupt) Close(ctx context.Context) error {
	return nil
}

// addresses at 6 to 70 + inputOffset.
func (di *digitalInterrupt) isInputCounter() bool {
	return di.Address >= di.inputOffset+inputWordToCounterOffset && di.Address < di.outputOffset
}

// addresses at 0 and 1 + inputOffset.
func (di *digitalInterrupt) isDigitalInput() bool {
	return di.Address == di.inputOffset || di.Address == di.inputOffset+1
}
