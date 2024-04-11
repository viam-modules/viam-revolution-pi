//go:build linux

// Package revolution_pi implements the Revolution Pi board GPIO pins.
package revolution_pi

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"

	commonpb "go.viam.com/api/common/v1"
	pb "go.viam.com/api/component/board/v1"
	"go.viam.com/rdk/components/board"
	"go.viam.com/rdk/grpc"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
)

type revolution_piBoard struct {
	resource.Named
	resource.TriviallyReconfigurable

	mu            sync.RWMutex
	logger        logging.Logger
	AnalogReaders []string
	GPIONames     []string

	controlChip             *gpioChip
	cancelCtx               context.Context
	cancelFunc              func()
	activeBackgroundWorkers sync.WaitGroup
}

func init() {
	resource.RegisterComponent(
		board.API,
		Model,
		resource.Registration[board.Board, *Config]{Constructor: newBoard})
}

func newBoard(
	ctx context.Context,
	_ resource.Dependencies,
	conf resource.Config,
	logger logging.Logger,
) (board.Board, error) {
	logger.Info("Starting revolution_pi Driver v0.0.5")
	devPath := filepath.Join("/dev", "piControl0")
	//nolint:gosec
	fd, err := os.OpenFile(devPath, os.O_RDWR, fs.FileMode(os.O_RDWR))
	if err != nil {
		err = fmt.Errorf("open chip %v failed: %w", devPath, err)
		return nil, err
	}
	cancelCtx, cancelFunc := context.WithCancel(context.Background())
	gpioChip := gpioChip{dev: devPath, logger: logger, fileHandle: fd}
	b := revolution_piBoard{
		Named:         conf.ResourceName().AsNamed(),
		logger:        logger,
		cancelCtx:     cancelCtx,
		cancelFunc:    cancelFunc,
		AnalogReaders: []string{},
		GPIONames:     []string{},
		controlChip:   &gpioChip,
		mu:            sync.RWMutex{},
	}

	err = b.controlChip.showDeviceList()
	if err != nil {
		return nil, err
	}

	return &b, nil
}

func (b *revolution_piBoard) AnalogReaderByName(name string) (board.AnalogReader, bool) {
	reader, err := b.controlChip.GetAnalogInput(name)
	if err != nil {
		b.logger.Error(err)
		return nil, false
	}
	b.logger.Infof("Analog Reader: %#v", reader)
	return reader, true
}

func (b *revolution_piBoard) DigitalInterruptByName(name string) (board.DigitalInterrupt, bool) {
	return nil, false // Digital interrupts aren't supported.
}

func (b *revolution_piBoard) AnalogReaderNames() []string {
	return nil
}

func (b *revolution_piBoard) DigitalInterruptNames() []string {
	return nil
}

func (b *revolution_piBoard) GPIOPinNames() []string {
	return nil
}

func (b *revolution_piBoard) GPIOPinByName(pinName string) (board.GPIOPin, error) {
	return b.controlChip.GetGPIOPin(pinName)
}

func (b *revolution_piBoard) Status(ctx context.Context, extra map[string]interface{}) (*commonpb.BoardStatus, error) {
	return &commonpb.BoardStatus{}, nil
}

func (b *revolution_piBoard) SetPowerMode(ctx context.Context, mode pb.PowerMode, duration *time.Duration) error {
	return grpc.UnimplementedError
}

func (b *revolution_piBoard) WriteAnalog(ctx context.Context, pin string, value int32, extra map[string]interface{}) error {
	return nil
}

func (b *revolution_piBoard) Close(ctx context.Context) error {
	b.mu.Lock()
	b.logger.Info("Closing RevPi board.")
	defer b.mu.Unlock()
	b.cancelFunc()
	err := b.controlChip.Close()
	if err != nil {
		return err
	}
	b.activeBackgroundWorkers.Wait()
	b.logger.Info("Board closed.")
	return nil
}

func (b *revolution_piBoard) DoCommand(ctx context.Context,
	req map[string]interface{},
) (map[string]interface{}, error) {
	resp := make(map[string]interface{})

	_, ok := req["getStatus"]
	if ok {
		err := b.controlChip.showDeviceList()
		if err != nil {
			return nil, err
		}
		resp["getStatus"] = "status ok"
	}

	return resp, nil
}
