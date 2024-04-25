//go:build linux

// Package genericlinux is for Linux boards, and this particular file is for digital interrupt pins
// using the ioctl interface, indirectly by way of mkch's gpio package.
package revolutionpi

import (
	"context"
	"errors"

	"go.viam.com/rdk/components/board"
	"go.viam.com/rdk/grpc"
)

const (
	inputModeOffset = 88
)

type digitalInterrupt struct {
	PinName      string // Variable name
	Address      uint16 // Address of the byte in the process image
	Length       uint16 // length of the variable in bits. Possible values are 1, 8, 16 and 32
	ControlChip  *gpioChip
	outputOffset uint16
	inputOffset  uint16
}

func (di *digitalInterrupt) initialize() error {
	var addressInputMode uint16
	if di.isInputCounter() {
		// read from the input mode byte to determine if the pin is configured for counter/interrupt mode

		// determine which address to check for the input mode
		addressInputMode = (di.Address - di.inputOffset - inputWordToCounterOffset) >> 2

	} else {
		return errors.New("pin is not a digital input pin")
	}
	b := make([]byte, 1)
	// all PWM pins use the same PWM frequency
	n, err := di.ControlChip.fileHandle.ReadAt(b, int64(di.inputOffset+inputModeOffset+addressInputMode))
	if err != nil {
		return err
	}
	if n != 1 {
		return errors.New("unable to read digital input pin configuration")
	}
	di.ControlChip.logger.Infof("Current Pin configuration: %#d", b)
	// check if the pin is configured as a counter
	if b[0] == 0 || b[0] == 4 {
		return errors.New("pin is not configured as a counter")
	}

	return nil
}

func (di *digitalInterrupt) Value(ctx context.Context, extra map[string]interface{}) (int64, error) {
	return 0, nil
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
