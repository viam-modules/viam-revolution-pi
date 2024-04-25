//go:build linux

// Package genericlinux is for Linux boards, and this particular file is for digital interrupt pins
// using the ioctl interface, indirectly by way of mkch's gpio package.
package revolutionpi

import (
	"context"

	"go.viam.com/rdk/components/board"
	"go.viam.com/rdk/grpc"
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
