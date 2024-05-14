//go:build linux

// Package revolutionpi implements the Revolution Pi board GPIO pins.
package revolutionpi

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"go.viam.com/rdk/components/encoder"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
)

type revolutionPiEncoder struct {
	resource.Named
	resource.TriviallyReconfigurable
	pin *digitalInterrupt
}

// Model is the model triplet for the rev-pi board.
var EncoderModel = resource.NewModel("viam-labs", "kunbus", "revolutionpi-encoder")

// // Config is the config for the rev-pi board.
// type Config struct {
// 	resource.TriviallyValidateConfig
// 	Attributes utils.AttributeMap `json:"attributes,omitempty"`
// }

func init() {
	resource.RegisterComponent(
		encoder.API,
		EncoderModel,
		resource.Registration[encoder.Encoder, *Config]{Constructor: newEncoder})
}

func newEncoder(
	ctx context.Context,
	_ resource.Dependencies,
	conf resource.Config,
	logger logging.Logger,
) (encoder.Encoder, error) {
	devPath := filepath.Join("/dev", "piControl0")
	devPath = filepath.Clean(devPath)
	fd, err := os.OpenFile(devPath, os.O_RDWR, fs.FileMode(os.O_RDWR))
	if err != nil {
		err = fmt.Errorf("open chip %v failed: %w", devPath, err)
		return nil, err
	}
	gpioChip := gpioChip{dev: devPath, logger: logger, fileHandle: fd}

	err = gpioChip.showDeviceList()
	if err != nil {
		return nil, err
	}
	name := "InputMode_5"
	pin := SPIVariable{strVarName: char32(name)}
	err = gpioChip.mapNameToAddress(&pin)
	if err != nil {
		return nil, err
	}

	enc, err := initializeDigitalInterrupt(pin, &gpioChip, true)
	if err != nil {
		return nil, err
	}
	if !enc.isEncoder {
		return nil, errors.New("address is not configured as an encoder")
	}
	return &revolutionPiEncoder{pin: enc}, nil
}

func (enc *revolutionPiEncoder) Position(ctx context.Context, positionType encoder.PositionType, extra map[string]interface{}) (float64, encoder.PositionType, error) {
	return 0, encoder.PositionTypeTicks, nil
}

func (enc *revolutionPiEncoder) ResetPosition(ctx context.Context, extra map[string]interface{}) error {
	return nil
}

func (enc *revolutionPiEncoder) Properties(ctx context.Context, extra map[string]interface{}) (encoder.Properties, error) {
	props := encoder.Properties{TicksCountSupported: true, AngleDegreesSupported: false}
	return props, nil
}

func (enc revolutionPiEncoder) DoCommand(ctx context.Context, req map[string]interface{}) (map[string]interface{}, error) {
	return nil, nil
}

func (enc revolutionPiEncoder) Close(ctx context.Context) error {
	return nil
}
