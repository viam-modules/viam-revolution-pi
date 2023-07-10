//go:build linux

// Package revolution_pi implements the Revolution Pi board GPIO pins.
package revolution_pi

import (
	"context"
	"errors"
	"unsafe"
)

type gpioPin struct {
	Name        string // Variable name
	Address     uint16 // Address of the byte in the process image
	BitPosition uint8  // 0-7 bit position, >= 8 whole byte
	Length      uint16 // length of the variable in bits. Possible values are 1, 8, 16 and 32
	ControlChip *gpioChip
}

func (pin *gpioPin) Set(ctx context.Context, high bool, extra map[string]interface{}) error {
	val := uint8(0)
	if high {
		val = uint8(1)
	}
	command := SPIValue{i16uAddress: pin.Address, i8uBit: pin.BitPosition, i8uValue: val}
	pin.ControlChip.ioCtl(uintptr(KB_SET_VALUE), uintptr(unsafe.Pointer(&command)))
	return nil
}

// Get gets the high/low state of the pin.
func (pin *gpioPin) Get(ctx context.Context, extra map[string]interface{}) (bool, error) {
	command := SPIValue{i16uAddress: pin.Address + (uint16(pin.BitPosition) / 8), i8uBit: pin.BitPosition % 8}
	err := pin.ControlChip.ioCtl(uintptr(KB_GET_VALUE), uintptr(unsafe.Pointer(&command)))
	ret := false
	if command.i8uValue == 1 {
		ret = true
	}
	return ret, err
}

// PWM gets the pin's given duty cycle.
func (pin *gpioPin) PWM(ctx context.Context, extra map[string]interface{}) (float64, error) {
	return 0, errors.New("not supported")
}

// SetPWM sets the pin to the given duty cycle.
func (pin *gpioPin) SetPWM(ctx context.Context, dutyCyclePct float64, extra map[string]interface{}) error {
	return errors.New("not supported")
}

// PWMFreq gets the PWM frequency of the pin.
func (pin *gpioPin) PWMFreq(ctx context.Context, extra map[string]interface{}) (uint, error) {
	return 0, errors.New("not supported")
}

// SetPWMFreq sets the given pin to the given PWM frequency. For Raspberry Pis,
// 0 will use a default PWM frequency of 800.
func (pin *gpioPin) SetPWMFreq(ctx context.Context, freqHz uint, extra map[string]interface{}) error {
	return errors.New("not supported")
}
