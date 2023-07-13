//go:build linux

// Package revolution_pi implements the Revolution Pi board GPIO pins.
package revolution_pi

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"unsafe"
)

type gpioPin struct {
	Name        string // Variable name
	Address     uint16 // Address of the byte in the process image
	BitPosition uint8  // 0-7 bit position, >= 8 whole byte
	Length      uint16 // length of the variable in bits. Possible values are 1, 8, 16 and 32
	ControlChip *gpioChip
}

// Get the memory address to use for modifying the PWM duty cycle
func (pin *gpioPin) getPwmAddress() uint16 {
	// The address for the pin is either 70 or 71,
	// the bit position is used to figure out the offset from the base address of 72 for PWM
	return pin.Address + 2 + (uint16(pin.BitPosition) % 8) + ((pin.Address - 70) * 7)
}

// Get the memory address to use for modifying the pin state (on/off)
func (pin *gpioPin) getGpioAddress() uint16 {
	return pin.Address + (uint16(pin.BitPosition) / 8)
}

// Set sets the state of the pin on or off
func (pin *gpioPin) Set(ctx context.Context, high bool, extra map[string]interface{}) error {
	val := uint8(0)
	if high {
		val = uint8(1)
	}
	// Because there could be a race in reading the byte with pin states, mutating,
	// and writing back, we can leverage the ioctl command to modify 1 bit
	command := SPIValue{i16uAddress: pin.Address, i8uBit: pin.BitPosition, i8uValue: val}
	pin.ControlChip.logger.Debugf("Command: %#v", command)
	err := pin.ControlChip.ioCtl(uintptr(KB_SET_VALUE), unsafe.Pointer(&command))
	if err != 0 {
		return err
	}
	return nil
}

// Get gets the high/low state of the pin.
func (pin *gpioPin) Get(ctx context.Context, extra map[string]interface{}) (bool, error) {
	pin.ControlChip.logger.Debugf("Reading from %v", pin.getGpioAddress())
	b := make([]byte, 1)
	n, err := pin.ControlChip.fileHandle.ReadAt(b, int64(pin.getGpioAddress()))
	pin.ControlChip.logger.Debugf("Read %#v bytes", b)
	if n != 1 {
		return false, fmt.Errorf("expected 1 byte, got %#v", b)
	}
	if err != nil {
		return false, err
	}
	if (b[0]>>pin.BitPosition)&1 == 1 {
		return true, nil
	} else {
		return false, nil
	}
}

// PWM gets the pin's given duty cycle.
func (pin *gpioPin) PWM(ctx context.Context, extra map[string]interface{}) (float64, error) {

	pwmAddress := pin.getPwmAddress()
	pin.ControlChip.logger.Debugf("Reading from %v", pwmAddress)
	b := make([]byte, 2)
	n, err := pin.ControlChip.fileHandle.ReadAt(b, int64(pwmAddress))
	pin.ControlChip.logger.Debugf("Read %#v bytes", b)
	if n != 2 {
		return 0, fmt.Errorf("expected 2 bytes, got %#v", b)
	}
	if err != nil {
		return 0, err
	}
	b[1] = 0x00
	val := binary.LittleEndian.Uint16(b)
	if val > 100 {
		pin.ControlChip.logger.Infof("got PWM duty cycle greater than 100")
	}
	return float64(val), nil
}

// SetPWM sets the pin to the given duty cycle.
func (pin *gpioPin) SetPWM(ctx context.Context, dutyCyclePct float64, extra map[string]interface{}) error {
	if dutyCyclePct > 100 {
		// Should we clamp or error?
		return errors.New("cannot set duty cycle greater than 100%")
	}
	if dutyCyclePct < 0 {
		return errors.New("cannot set duty cycle less than 0%")
	}

	// We actually need to enable PWM for the pin, we can't just set the PWM value
	// Much like in Set, we can't modify a single bit using the regular read/write in file stream
	// so we have to use the ioctl command to modify just a single bit
	command := SPIValue{i16uAddress: 110 + (pin.Address - 70), i8uBit: pin.BitPosition, i8uValue: 1}
	pin.ControlChip.logger.Infof("Command: %#v", command)
	syscallErr := pin.ControlChip.ioCtl(uintptr(KB_SET_VALUE), unsafe.Pointer(&command))
	if syscallErr != 0 {
		return fmt.Errorf("error turning on pwm for output: %v", syscallErr)
	}

	pwmAddress := pin.getPwmAddress()
	pin.ControlChip.logger.Infof("Writing to %v", pwmAddress)
	b := make([]byte, 2)
	binary.LittleEndian.PutUint16(b, uint16(dutyCyclePct))
	b = b[:1]
	n, err := pin.ControlChip.fileHandle.WriteAt(b, int64(pwmAddress))
	pin.ControlChip.logger.Infof("Wrote %#v byte(s), n: %v", b, n)
	if n < 1 || n > 1 {
		return fmt.Errorf("expected 1 byte(s), got %#v", b)
	}
	return err
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
