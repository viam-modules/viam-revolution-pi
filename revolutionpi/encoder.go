//go:build linux

// Package revolutionpi implements the Revolution Pi board GPIO pins.
package revolutionpi

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync/atomic"

	"go.viam.com/rdk/components/encoder"
	"go.viam.com/rdk/grpc"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	"go.viam.com/utils"
)

// revolutionPiEncoder wraps a digital interrupt pin with the Encoder interface.
type revolutionPiEncoder struct {
	resource.Named
	resource.AlwaysRebuild
	pin     *counterPin
	zeroPos atomic.Int32
}

// EncoderModel is the model triplet for the rev-pi board encoder.
var EncoderModel = resource.NewModel("viam", "kunbus", "revolutionpi-encoder")

// EncoderConfig is the config for the rev-pi board encoder.
type EncoderConfig struct {
	Name string `json:"pin_name"`
}

func init() {
	resource.RegisterComponent(
		encoder.API,
		EncoderModel,
		resource.Registration[encoder.Encoder, *EncoderConfig]{Constructor: newEncoder})
}

// Validate validates the EncoderConfig.
func (cfg *EncoderConfig) Validate(path string) ([]string, error) {
	if cfg.Name == "" {
		return nil, utils.NewConfigValidationFieldRequiredError(path, "pin_name")
	}
	return []string{}, nil
}

func newEncoder(
	ctx context.Context,
	_ resource.Dependencies,
	conf resource.Config,
	logger logging.Logger,
) (encoder.Encoder, error) {
	svcConfig, err := resource.NativeConfig[*EncoderConfig](conf)
	if err != nil {
		return nil, err
	}
	devPath := filepath.Clean(filepath.Join("/dev", "piControl0"))
	fd, err := os.OpenFile(devPath, os.O_RDWR, fs.FileMode(os.O_RDWR))
	if err != nil {
		err = fmt.Errorf("open chip %v failed: %w", devPath, err)
		return nil, err
	}
	chip := gpioChip{dev: devPath, logger: logger, fileHandle: fd}

	err = chip.showDeviceList()
	if err != nil {
		return nil, err
	}
	name := svcConfig.Name
	pin := SPIVariable{strVarName: char32(name)}
	err = chip.mapNameToAddress(&pin)
	if err != nil {
		return nil, err
	}

	enc, err := initializeDigitalInterrupt(pin, &chip, true)
	if err != nil {
		return nil, err
	}

	return &revolutionPiEncoder{Named: conf.ResourceName().AsNamed(), pin: enc}, nil
}

func (enc *revolutionPiEncoder) Position(ctx context.Context, positionType encoder.PositionType,
	extra map[string]interface{},
) (float64, encoder.PositionType, error) {
	pos, err := enc.pin.Value()
	if err != nil {
		return 0, encoder.PositionTypeTicks, err
	}
	// rev pi encoder values are int32, but Value() returns uint32. we cast the pos to int32 to fix this
	signedPos := int32(pos) - enc.zeroPos.Load()

	// encoder api expects float64
	return float64(signedPos), encoder.PositionTypeTicks, nil
}

func (enc *revolutionPiEncoder) ResetPosition(ctx context.Context, extra map[string]interface{}) error {
	pos, err := enc.pin.Value()
	if err != nil {
		return err
	}
	enc.zeroPos.Store(int32(pos))
	return nil
}

func (enc *revolutionPiEncoder) Properties(ctx context.Context, extra map[string]interface{}) (encoder.Properties, error) {
	return encoder.Properties{TicksCountSupported: true, AngleDegreesSupported: false}, nil
}

func (enc *revolutionPiEncoder) DoCommand(ctx context.Context, req map[string]interface{}) (map[string]interface{}, error) {
	return nil, grpc.UnimplementedError
}

func (enc *revolutionPiEncoder) Close(ctx context.Context) error {
	return enc.pin.controlChip.Close()
}
