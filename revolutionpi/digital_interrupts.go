//go:build linux

// Package revolutionpi implements the Revolution Pi board GPIO pins.
package revolutionpi

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"go.viam.com/rdk/components/board"
)

const (
	inputModeOffset = 88
)

type digitalInterrupt struct {
	pinName          string // Variable name
	address          uint16 // address of the byte in the process image
	length           uint16 // length of the variable in bits. Possible values are 1, 8, 16 and 32
	bitPosition      uint8  // 0-7 bit position, >= 8 whole byte, only used if the digital input pin is given
	controlChip      *gpioChip
	outputOffset     uint16
	inputOffset      uint16
	enabled          bool
	isEncoder        bool
	interruptAddress uint16
}

type diWrapper struct {
	pin *digitalInterrupt
}

func initializeDigitalInterrupt(pin SPIVariable, g *gpioChip, isEncoder bool) (*digitalInterrupt, error) {
	di := digitalInterrupt{
		pinName: str32(pin.strVarName), address: pin.i16uAddress,
		length: pin.i16uLength, bitPosition: pin.i8uBit, controlChip: g,
	}
	g.logger.Debugf("setting up digital interrupt pin: %v", di)
	dio, err := findDevice(di.address, g.dioDevices)
	if err != nil {
		return &digitalInterrupt{}, err
	}
	// store the input & output offsets of the board for quick reference
	di.outputOffset = dio.i16uOutputOffset
	di.inputOffset = dio.i16uInputOffset

	var addressInputMode uint16

	// read from the input mode byte to determine if the pin is configured for counter/interrupt mode
	// determine which address to check for the input mode based on which pin was given in the request
	switch {
	case di.isInputCounter():
		addressInputMode = (di.address - di.inputOffset - inputWordToCounterOffset) >> 2

		// record the address for the interrupt
		di.interruptAddress = di.address
	case di.isDigitalInput():
		addressInputMode = uint16(di.bitPosition)
		if di.address > di.inputOffset { // This is the second set of input pins, so move the offset over
			addressInputMode += 8
		}
		di.interruptAddress = di.inputOffset + inputWordToCounterOffset + addressInputMode*4
	default:
		return &digitalInterrupt{}, errors.New("pin is not a digital input pin")
	}

	b := make([]byte, 1)
	// read from the input mode addresses to see if the pin is configured for interrupts
	n, err := di.controlChip.fileHandle.ReadAt(b, int64(di.inputOffset+inputModeOffset+addressInputMode))
	if err != nil {
		return &digitalInterrupt{}, err
	}
	if n != 1 {
		return &digitalInterrupt{}, errors.New("unable to read digital input pin configuration")
	}
	di.controlChip.logger.Debugf("Current Pin configuration: %#d", b)

	// check if the pin is configured as a counter
	// b[0] == 0 means the interrupt is disabled, b[0] == 3 means the pin is configured for encoder mode
	if b[0] == 0 || (b[0] == 3 && !isEncoder) {
		return &digitalInterrupt{}, fmt.Errorf("pin %s is not configured as a counter", di.pinName)
	}

	di.enabled = true
	di.isEncoder = b[0] == 3

	return &di, nil
}

func (di *diWrapper) Value(ctx context.Context, extra map[string]interface{}) (int64, error) {
	return di.pin.Value()
}

// Note: The revolution pi only supports uint32 counters, while the Value API expects int64.
func (di *digitalInterrupt) Value() (int64, error) {
	if !di.enabled {
		return 0, fmt.Errorf("cannot get digital interrupt value, pin %s is not configured as an interrupt", di.pinName)
	}
	di.controlChip.logger.Debugf("Reading from %d, length: 4 byte(s)", di.interruptAddress)
	b := make([]byte, 4)
	n, err := di.controlChip.fileHandle.ReadAt(b, int64(di.interruptAddress))
	if err != nil {
		return 0, err
	}
	di.controlChip.logger.Debugf("Read %#v bytes", b)
	if n != 4 {
		return 0, fmt.Errorf("expected 4 bytes, got %#v", b)
	}
	val := binary.LittleEndian.Uint32(b)
	return int64(val), nil
}

func (di *diWrapper) Name() string {
	return di.pin.pinName
}

func (di *diWrapper) RemoveCallback(c chan board.Tick) {}

func (di *diWrapper) Close(ctx context.Context) error {
	return nil
}

// addresses at 6 to 70 + inputOffset.
func (di *digitalInterrupt) isInputCounter() bool {
	return di.address >= di.inputOffset+inputWordToCounterOffset && di.address < di.outputOffset
}

// addresses at 0 and 1 + inputOffset.
func (di *digitalInterrupt) isDigitalInput() bool {
	return di.address == di.inputOffset || di.address == di.inputOffset+1
}
