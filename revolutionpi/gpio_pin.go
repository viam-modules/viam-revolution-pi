//go:build linux

// Package revolutionpi implements the Revolution Pi board GPIO pins.
package revolutionpi

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"unsafe"
)

const (
	// address offsets for DIO boards. See DIO documentation for more information
	// https://revolutionpi.com/en/tutorials/overview-revpi-io-modules
	outputPWMActiveOffset    = 110 // address offset for reading PWM duty cycles for pins
	outputPWMFrequencyOffset = 112 // address offset for reading a PWM frequency
	outputWordToPWMOffset    = 2   // address offset between digital outputs and PWM addresses
	inputWordToCounterOffset = 6   // address offset for input counter/interrupt addresses
	dioMemoryOffset          = 88  // address offset for memory addresses
)

type gpioPin struct {
	Name         string // Variable name
	Address      uint16 // Address of the byte in the process image
	BitPosition  uint8  // 0-7 bit position, >= 8 whole byte
	Length       uint16 // length of the variable in bits. Possible values are 1, 8, 16 and 32
	ControlChip  *gpioChip
	pwmMode      bool
	initialized  bool
	outputOffset uint16
	inputOffset  uint16
}

func (pin *gpioPin) initialize() error {
	var err error
	val := false

	// for output gpio pins pwm can be enabled, so we should check for that
	if pin.isDigitalOutput() {
		// if the normal gpio output is given, use the bit position to check if we are in pwm mode.
		// We also need to determine which address to check.
		pwmActiveAddress := int64(pin.Address - pin.outputOffset + pin.inputOffset + outputPWMActiveOffset)
		val, err = pin.ControlChip.getBitValue(pwmActiveAddress, pin.BitPosition)
		if err != nil {
			return err
		}
	} else if pin.isOutputPWM() {
		// we want to read a bit from OutputPWMActive WORD to see if pwm is enabled,
		// so we convert the pin address into the matching bits, where PWM_1 corresponds to bit 0.
		// Output pins start at pin.outputOffset+2, so we can subtract pin address by that amount to get the correct bit
		pwmActiveBitPosition := uint8(pin.Address - pin.outputOffset - outputWordToPWMOffset) // between 0 and 16
		pwmActiveAddress := int64(pin.inputOffset + outputPWMActiveOffset + uint16(pwmActiveBitPosition>>3))
		val, err = pin.ControlChip.getBitValue(pwmActiveAddress, pwmActiveBitPosition%8)
		if err != nil {
			return err
		}
	}

	pin.pwmMode = val
	pin.initialized = true

	pin.ControlChip.logger.Debugf("Pin initialized: %#v", pin)
	return nil
}

// Get the memory address to use for modifying the PWM duty cycle. This should Only be used when a PWM
// request is made to a GPIO output pin.
func (pin *gpioPin) getPwmAddress() uint16 {
	// The address for the Output Word pin is either 0 or 1 + outputOffset. Multiply by 7 to move to the correct address.
	firstOrSecondHalf := 7 * (pin.Address - pin.outputOffset)
	// the bit position then gets used to determine which PWM pin should be used
	return pin.Address + outputWordToPWMOffset + firstOrSecondHalf + (uint16(pin.BitPosition))
}

// Get the memory address to use for modifying the pin state (on/off).
func (pin *gpioPin) getGpioAddress() uint16 {
	switch {
	// if a PWM pin is given for GPIO behaviors
	case pin.isOutputPWM():
		// subtract the offsets from the Address from the PWM address,
		// then shift by 3 to get the 0 or 1 address
		return pin.outputOffset + (pin.Address-outputWordToPWMOffset-pin.outputOffset)>>3
	// if an Input Counter pin is given for GPIO behaviors
	case pin.isInputCounter():
		// subtract the offsets from the Address of the input counter to be a value from 0 to 63,
		// then shift by 5 to get the 0 or 1 address
		return pin.inputOffset + (pin.Address-pin.inputOffset-inputWordToCounterOffset)>>5
	// by default we are a GPIO pin
	default:
		return pin.Address
	}
}

// Set sets the state of the pin on or off.
func (pin *gpioPin) Set(ctx context.Context, high bool, extra map[string]interface{}) error {
	if !pin.initialized {
		return errors.New("pin not initialized")
	}

	val := uint8(0)
	if high {
		val = uint8(1)
	}

	// Error if we are not a pin that can support GPIO Outputs
	if !pin.isOutputPWM() && !pin.isDigitalOutput() {
		return fmt.Errorf("cannot set pin state, Pin %s is not a digital output pin", pin.Name)
	}

	// error if PWM is enabled for the pin in question
	if pin.pwmMode {
		return fmt.Errorf("cannot set pin state, Pin %s is configured as PWM", pin.Name)
	}

	gpioAddress := pin.getGpioAddress()
	gpioBit := pin.BitPosition

	// if someone used the PWM pin name, get the bit for the GPIO output
	if !pin.isDigitalOutput() {
		gpioBit = uint8(pin.Address-outputWordToPWMOffset-pin.outputOffset) % 8
	}

	// Because there could be a race in reading the byte with pin states, mutating,
	// and writing back, we can leverage the ioctl command to modify 1 bit
	command := SPIValue{i16uAddress: gpioAddress, i8uBit: gpioBit, i8uValue: val}
	pin.ControlChip.logger.Debugf("Command: %#v", command)
	//nolint:gosec
	err := pin.ControlChip.ioCtl(uintptr(kbSetValue), unsafe.Pointer(&command))
	if err != 0 {
		return err
	}
	return nil
}

// Get gets the high/low state of the pin.
func (pin *gpioPin) Get(ctx context.Context, extra map[string]interface{}) (bool, error) {
	if !pin.initialized {
		return false, errors.New("pin not initialized")
	}
	if pin.pwmMode {
		return false, fmt.Errorf("cannot get pin state, Pin %s is configured as PWM", pin.Name)
	}

	gpioAddress := pin.getGpioAddress()
	gpioBit := pin.BitPosition

	if pin.isOutputPWM() {
		// if someone used the PWM pin name, get the bit for the GPIO pin
		gpioBit = uint8((pin.Address-outputWordToPWMOffset)%pin.outputOffset) % 8
	} else if pin.isInputCounter() {
		// if someone used the Counter pin name, get the bit for the GPIO pin
		// get the address into 4 byte chunks, then mod 8 for the bit location
		gpioBit = uint8(pin.Address-inputWordToCounterOffset-pin.inputOffset) >> 2 % 8
	}

	pin.ControlChip.logger.Debugf("Reading from Address %d, bit %d", gpioAddress, gpioBit)

	return pin.ControlChip.getBitValue(int64(gpioAddress), gpioBit)
}

// PWM gets the pin's given duty cycle.
func (pin *gpioPin) PWM(ctx context.Context, extra map[string]interface{}) (float64, error) {
	if !pin.initialized {
		return 0, errors.New("pin not initialized")
	}
	if !pin.isOutputPWM() && !pin.isDigitalOutput() {
		return 0, fmt.Errorf("cannot get PWM, Pin %s is not a PWM pin", pin.Name)
	}

	// if the pin isn't configured for PWM mode, throw an error
	if !pin.pwmMode {
		return 0, fmt.Errorf("cannot get PWM, Pin %s is not configured for PWM", pin.Name)
	}

	pwmAddress := pin.Address
	if pin.isDigitalOutput() {
		pwmAddress = pin.getPwmAddress()
	}

	b := make([]byte, 2)
	n, err := pin.ControlChip.fileHandle.ReadAt(b, int64(pwmAddress))
	pin.ControlChip.logger.Debugf("Read %#d bytes", b)
	if n != 2 {
		return 0, fmt.Errorf("expected 2 bytes, got %#v", b)
	}
	if err != nil {
		return 0, err
	}
	b[1] = 0x00
	val := binary.LittleEndian.Uint16(b)
	if val > 100 {
		pin.ControlChip.logger.Warn("got PWM duty cycle greater than 100")
	}
	return float64(val) / 100, nil
}

// SetPWM sets the pin to the given duty cycle.
func (pin *gpioPin) SetPWM(ctx context.Context, dutyCyclePct float64, extra map[string]interface{}) error {
	if !pin.initialized {
		return errors.New("pin not initialized")
	}
	if !pin.isOutputPWM() && !pin.isDigitalOutput() {
		return fmt.Errorf("cannot set PWM, Pin %s is not a PWM pin", pin.Name)
	}

	// if the pin isn't configured for PWM mode, throw an error
	if !pin.pwmMode {
		return fmt.Errorf("cannot set PWM, Pin %s is not configured for PWM", pin.Name)
	}

	// convert from 0-1 to 0-100
	dutyCyclePct = 100 * dutyCyclePct
	if dutyCyclePct > 100 {
		// Should we clamp or error?
		return errors.New("cannot set duty cycle greater than 100%")
	}
	if dutyCyclePct < 0 {
		return errors.New("cannot set duty cycle less than 0%")
	}

	pwmAddress := pin.Address
	if pin.isDigitalOutput() {
		pwmAddress = pin.getPwmAddress()
	}

	b := make([]byte, 2)
	binary.LittleEndian.PutUint16(b, uint16(dutyCyclePct))
	b = b[:1]
	err := pin.ControlChip.writeValue(int64(pwmAddress), b)
	return err
}

// PWMFreq gets the PWM frequency of the pin.
func (pin *gpioPin) PWMFreq(ctx context.Context, extra map[string]interface{}) (uint, error) {
	if !pin.initialized {
		return 0, errors.New("pin not initialized")
	}
	if !pin.isOutputPWM() && !pin.isDigitalOutput() {
		return 0, fmt.Errorf("cannot get PWM Frequency, Pin %s is not a PWM pin", pin.Name)
	}

	b := make([]byte, 1)
	// all PWM pins use the same PWM frequency
	n, err := pin.ControlChip.fileHandle.ReadAt(b, int64(pin.inputOffset+outputPWMFrequencyOffset))
	if err != nil {
		return 0, err
	}
	if n != 1 {
		return 0, errors.New("unable to read PWM Frequency")
	}
	pin.ControlChip.logger.Debugf("Current frequency step size: %#d", b)

	return stepSizeToFreq(b), nil
}

// stepSizeToFreq returns the frequency based on the step size returned from the outputPWMFrequency address
// see documentation for more information.
func stepSizeToFreq(step []byte) uint {
	// b only has one byte
	switch step[0] {
	case 1:
		return 40
	case 2:
		return 80
	case 4:
		return 160
	case 5:
		return 200
	case 10:
		return 400
	}
	return 0
}

// SetPWMFreq sets the given pin to the given PWM frequency. For the Rev-Pi this must be configured in PiCtory.
func (pin *gpioPin) SetPWMFreq(ctx context.Context, freqHz uint, extra map[string]interface{}) error {
	if !pin.initialized {
		return errors.New("pin not initialized")
	}

	return errors.New("PWM Frequency must be set in PiCtory")
}

// pins at 70 or 71 + inputOffset.
func (pin *gpioPin) isDigitalOutput() bool {
	return pin.Address == pin.outputOffset || pin.Address == pin.outputOffset+1
}

// pins with an offset of 72 to 87 + inputOffset.
func (pin *gpioPin) isOutputPWM() bool {
	return pin.Address >= pin.outputOffset+outputWordToPWMOffset && pin.Address < pin.inputOffset+dioMemoryOffset
}

// pins at 6 to 70 + inputOffset.
func (pin *gpioPin) isInputCounter() bool {
	return pin.Address >= pin.inputOffset+inputWordToCounterOffset && pin.Address < pin.outputOffset
}
