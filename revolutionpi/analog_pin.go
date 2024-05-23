//go:build linux

// Package revolutionpi implements the Revolution Pi board GPIO pins.
package revolutionpi

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"

	"go.viam.com/rdk/components/board"
)

const (
	analogInputMemAddress = 24
)

type analogPin struct {
	Name         string // Variable name
	Address      uint16 // Address of the byte in the process image
	Length       uint16 // length of the variable in bits. Possible values are 1, 8, 16 and 32
	ControlChip  *gpioChip
	outputOffset uint16
	inputOffset  uint16
	info         analogInfo
}

type analogInfo struct {
	min       int
	max       int
	isCurrent bool
}

func initializeAnalogPin(pin SPIVariable, g *gpioChip) (*analogPin, error) {
	analogPin := analogPin{Name: str32(pin.strVarName), Address: pin.i16uAddress, Length: pin.i16uLength, ControlChip: g}
	aio, err := findDevice(analogPin.Address, g.aioDevices)
	if err != nil {
		analogPin.ControlChip.logger.Debug("pin is not from a supported GPIO board")
		return nil, err
	}

	// store the input & output offsets of the board for quick reference
	analogPin.outputOffset = aio.i16uOutputOffset
	analogPin.inputOffset = aio.i16uInputOffset

	if analogPin.isAnalogInput() {
		analogInputNumber := (analogPin.Address - analogPin.inputOffset) / 2                     // results in 0, 1, 2, or 3
		inputRangeAddress := analogInputNumber*7 + analogInputMemAddress + analogPin.inputOffset // results in pin 24, 31, 38, or 45
		bufInputRange := make([]byte, 1)
		n, err := analogPin.ControlChip.fileHandle.ReadAt(bufInputRange, int64(inputRangeAddress))
		if err != nil {
			return nil, fmt.Errorf("failed to read input range for analog pin %s", analogPin.Name)
		}
		if n != 1 {
			return nil, fmt.Errorf("expected 1 byte, got %#v", bufInputRange)
		}
		analogPin.info, err = getAnalogInputRange(bufInputRange[0])
		if err != nil {
			return nil, err
		}
	} else if analogPin.isAnalogOutput() {
		// check to see if analog output is enabled
		outputRangeAddress := analogPin.inputOffset + 69
		// use the corresponding analog OutputRange pin to check if the analog output is enabled
		if analogPin.Address == analogPin.outputOffset+2 {
			outputRangeAddress = analogPin.inputOffset + 79
		}
		bufOutputRange := make([]byte, 1)
		n, err := analogPin.ControlChip.fileHandle.ReadAt(bufOutputRange, int64(outputRangeAddress))
		if err != nil {
			return nil, err
		}
		if n != 1 {
			return nil, fmt.Errorf("unable to determine if pin %s is configured for analog write", analogPin.Name)
		}
		analogPin.ControlChip.logger.Debugf("outputRange Value: %d", bufOutputRange)
		analogPin.info, err = getAnalogOutputRange(bufOutputRange[0], analogPin.Name)
		if err != nil {
			return nil, err
		}
	}
	return &analogPin, nil
}

func (pin *analogPin) Read(ctx context.Context, extra map[string]interface{}) (board.AnalogValue, error) {
	if !pin.isAnalogInput() {
		return board.AnalogValue{}, fmt.Errorf("cannot ReadAnalog, pin %s is not an analog input pin", pin.Name)
	}
	pin.ControlChip.logger.Debugf("Reading from %v, length: %v byte(s)", pin.Address, pin.Length/8)
	b := make([]byte, pin.Length/8)
	n, err := pin.ControlChip.fileHandle.ReadAt(b, int64(pin.Address))
	pin.ControlChip.logger.Debugf("Read %#v bytes", b)
	if n != 2 {
		return board.AnalogValue{}, fmt.Errorf("expected 2 bytes, got %#v", b)
	}
	if err != nil {
		return board.AnalogValue{}, err
	}
	val := binary.LittleEndian.Uint16(b)
	// NOTE: we currently assume that the input multiplier, divisor, and offset have not been modified
	// the min and max values will change if a user modifies these.
	// step size converts mV -> V and micro Amps -> mA
	analogVal := board.AnalogValue{Value: int(val), Min: float32(pin.info.min), Max: float32(pin.info.max), StepSize: 0.001}
	return analogVal, nil
}

func (pin *analogPin) Close(ctx context.Context) error {
	// There is nothing to close with respect to individual analog _reader_ pins
	return nil
}

func (pin *analogPin) Write(ctx context.Context, value int, extra map[string]interface{}) error {
	pin.ControlChip.logger.Debugf("Analog: %#v", pin)
	if !pin.isAnalogOutput() {
		return fmt.Errorf("cannot Write to Analog, pin %s is not an analog output pin", pin.Name)
	}

	// validate the requested value is within the range of the pin.
	// NOTE: we currently assume that the output multiplier, divisor, and offset have not been modified
	if value > pin.info.max || value < pin.info.min {
		return fmt.Errorf("value of %v is not within expected range (%v to %v)", value, pin.info.min, pin.info.max)
	}

	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, int32(value))
	if err != nil {
		return err
	}
	return pin.ControlChip.writeValue(int64(pin.Address), buf.Bytes())
}

// Analog output pins are located at address 0 or 2 + outputOffset.
func (pin *analogPin) isAnalogOutput() bool {
	return pin.Address == pin.outputOffset || pin.Address == pin.outputOffset+2
}

// Analog input pins are located at address 0-7 + inputOffset.
func (pin *analogPin) isAnalogInput() bool {
	return pin.Address >= pin.inputOffset && pin.Address < pin.inputOffset+8
}

func getAnalogOutputRange(val byte, name string) (analogInfo, error) {
	switch val {
	case 0:
		return analogInfo{}, fmt.Errorf("pin %s is not configured for analog write", name)
	case 1: // 0 to 5000 mV
		return analogInfo{min: 0, max: 5000, isCurrent: false}, nil
	case 2: // 0 to 10000 mV
		return analogInfo{min: 0, max: 10000, isCurrent: false}, nil
	case 3: // -5000 to 5000 mV
		return analogInfo{min: -5000, max: 5000, isCurrent: false}, nil
	case 4: // -10000 to 10000 mV
		return analogInfo{min: -10000, max: 10000, isCurrent: false}, nil
	case 5: // 0 to 5500 mV
		return analogInfo{min: 0, max: 5500, isCurrent: false}, nil
	case 6: // 0 to 11000 mV
		return analogInfo{min: 0, max: 11000, isCurrent: false}, nil
	case 7: // -5500 to 5500 mV
		return analogInfo{min: -5500, max: 5500, isCurrent: false}, nil
	case 8: // -11000 to 11000 mV
		return analogInfo{min: -11000, max: 11000, isCurrent: false}, nil
	case 9: // 4 to 20 mA
		return analogInfo{min: 4000, max: 20000, isCurrent: true}, nil
	case 10: // 0 to 20 mA
		return analogInfo{min: 0, max: 20000, isCurrent: true}, nil
	case 11: // 0 to 24 mA
		return analogInfo{min: 0, max: 24000, isCurrent: true}, nil
	default:
		return analogInfo{}, fmt.Errorf("invalid output range received, got %v", val)
	}
}

func getAnalogInputRange(val byte) (analogInfo, error) {
	switch val {
	case 1: // -10000 to 10000 mV
		return analogInfo{min: -10000, max: 10000, isCurrent: false}, nil
	case 2: // 0 to 10000 mV
		return analogInfo{min: 0, max: 10000, isCurrent: false}, nil
	case 3: // 0 to 5000 mV
		return analogInfo{min: 0, max: 5000, isCurrent: false}, nil
	case 4: // -5000 to 5000 mV
		return analogInfo{min: -5000, max: 5000, isCurrent: false}, nil
	case 5: // 0 to 20 mA
		return analogInfo{min: 0, max: 20000, isCurrent: true}, nil
	case 6: // 0 to 24 mA
		return analogInfo{min: 0, max: 24000, isCurrent: true}, nil
	case 7: // 4 to 20 mA
		return analogInfo{min: 4000, max: 20000, isCurrent: true}, nil
	case 8: // -25 to 25 mA
		return analogInfo{min: -25000, max: 25000, isCurrent: true}, nil
	default:
		return analogInfo{}, fmt.Errorf("invalid input range received, got %v", val)
	}
}
