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

const (
	outputPWMActiveOffset    = 110 // offset for reading a PWM frequency from a DIO
	outputPWMFrequencyOffset = 112
	outputWordToPWMOffset    = 2
	inputWordToCounterOffset = 6
	dioMemoryOffset          = 88
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
	dio, err := pin.ControlChip.findDIODevice(pin.Address)
	if err != nil {
		pin.ControlChip.logger.Debug("pin is not from the DIO")
		return err
	}

	// if the requested pin is checking the Output WORD. The WORD takes up to 2 bytes
	pin.outputOffset = dio.i16uOutputOffset
	pin.inputOffset = dio.i16uInputOffset

	val := false

	// for output gpio pins pwm can be enabled, so we should check for that
	if pin.isOutputWord() {
		// if the normal gpio output is given, use the bit position to check if we are in pwm mode
		val, err = pin.ControlChip.getBitValue(int64(dio.i16uInputOffset+outputPWMActiveOffset+pin.Address%dio.i16uOutputOffset), pin.BitPosition)
		if err != nil {
			return err
		}
	} else if pin.isOutputPWM() {
		// we want to read a bit from OutputPWMActive WORD to see if pwm is enabled,
		// so we convert the pin address into the matching bits, where PWM_1 corresponds to bit 0.
		// Output pins start at dio.i16uOutputOffset+2, so we can subtract pin address by that amount to get the correct bit
		pwmActiveBitPosition := uint8(pin.Address - dio.i16uOutputOffset - outputWordToPWMOffset)
		val, err = pin.ControlChip.getBitValue(int64(dio.i16uInputOffset+outputPWMActiveOffset+uint16(pwmActiveBitPosition/8)), pwmActiveBitPosition%8)
		if err != nil {
			return err
		}
	}

	pin.pwmMode = val
	pin.initialized = true

	pin.ControlChip.logger.Debugf("Pin initialized: %#v", pin)
	return nil
}

// Get the memory address to use for modifying the PWM duty cycle.
func (pin *gpioPin) getPwmAddress() uint16 {
	// The address for the pin is either 0 or 1 + the offset,
	// the bit position is used to figure out the offset from the base address of 2 + offset for PWM
	return pin.Address + 7*pin.Address%pin.outputOffset + outputWordToPWMOffset + (uint16(pin.BitPosition))
}

// Get the memory address to use for modifying the pin state (on/off).
func (pin *gpioPin) getGpioAddress() uint16 {
	switch {
	case pin.isOutputPWM():
		return pin.outputOffset + (pin.Address-outputWordToPWMOffset)%pin.outputOffset/8
	case pin.isInputCounter():
		// shift the input counter to be a value from 0 to 63, then divide by 32 to get the 0 or 1 address
		return pin.inputOffset + (pin.Address-pin.inputOffset-inputWordToCounterOffset)/32
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

	if !pin.isOutputPWM() && !pin.isOutputWord() {
		return fmt.Errorf("cannot set pin state, Pin %s is not a digital output pin", pin.Name)
	}

	// error if using PWM mode
	if pin.pwmMode {
		return fmt.Errorf("cannot set pin state, Pin %s is configured as PWM", pin.Name)
	}

	gpioAddress := pin.getGpioAddress()
	gpioBit := pin.BitPosition

	// if someone used the PWM pin name, get the GPIO address of the pin
	if !pin.isOutputWord() {
		gpioBit = uint8((pin.Address-2)%pin.outputOffset) % 8
	}

	// Because there could be a race in reading the byte with pin states, mutating,
	// and writing back, we can leverage the ioctl command to modify 1 bit
	command := SPIValue{i16uAddress: gpioAddress, i8uBit: gpioBit, i8uValue: val}
	pin.ControlChip.logger.Debugf("Command: %#v", command)
	//nolint:gosec
	err := pin.ControlChip.ioCtl(uintptr(KB_SET_VALUE), unsafe.Pointer(&command))
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
		// if someone used the PWM pin name, get the get the bit for the pin
		gpioBit = uint8((pin.Address-outputWordToPWMOffset)%pin.outputOffset) % 8
	} else if pin.isInputCounter() {
		// if someone used the Counter pin name, get the get the bit for the pin
		// get the address into 4 byte chunks, then mod 8 for the bit location
		gpioBit = uint8((pin.Address-inputWordToCounterOffset)%pin.inputOffset) / 4 % 8
	}

	pin.ControlChip.logger.Debugf("Reading from %d", gpioAddress)
	pin.ControlChip.logger.Debugf("Reading from bit %d", gpioBit)

	return pin.ControlChip.getBitValue(int64(gpioAddress), gpioBit)
}

// PWM gets the pin's given duty cycle.
func (pin *gpioPin) PWM(ctx context.Context, extra map[string]interface{}) (float64, error) {
	if !pin.initialized {
		return 0, errors.New("pin not initialized")
	}
	if !pin.isOutputPWM() && !pin.isOutputWord() {
		return 0, fmt.Errorf("cannot get PWM, Pin %s is not a PWM pin", pin.Name)
	}
	// if the pin isn't in PWM mode, throw an error
	if !pin.pwmMode {
		return 0, fmt.Errorf("cannot get PWM, Pin %s is not configured for PWM", pin.Name)
	}

	pwmAddress := pin.Address
	if pin.isOutputWord() {
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
	if !pin.isOutputPWM() && !pin.isOutputWord() {
		return fmt.Errorf("cannot set PWM, Pin %s is not a PWM pin", pin.Name)
	}

	// if the pin isn't in PWM mode, throw an error
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
	if pin.isOutputWord() {
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
	if !pin.isOutputPWM() && !pin.isOutputWord() {
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
func (pin *gpioPin) isOutputWord() bool {
	return pin.Address == pin.outputOffset || pin.Address == pin.outputOffset+1
}

// pins with an offset of 72 to 87 + inputOffset.
func (pin *gpioPin) isOutputPWM() bool {
	return pin.Address > pin.outputOffset+outputWordToPWMOffset && pin.Address < pin.inputOffset+dioMemoryOffset
}

// pins at 0 or 1 + inputOffset.
func (pin *gpioPin) isInputWord() bool {
	return pin.Address == pin.inputOffset || pin.Address == pin.inputOffset+1
}

// pins at 6 to 70 + inputOffset.
func (pin *gpioPin) isInputCounter() bool {
	return pin.Address >= pin.inputOffset+inputWordToCounterOffset && pin.Address < pin.outputOffset
}
