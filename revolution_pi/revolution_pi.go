//go:build linux

// Package revolution_pi implements the Revolution Pi board GPIO pins.
package revolution_pi

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/edaniels/golog"
	commonpb "go.viam.com/api/common/v1"
	pb "go.viam.com/api/component/board/v1"
	"go.viam.com/rdk/components/board"
	"go.viam.com/rdk/grpc"
	"go.viam.com/rdk/resource"
	"golang.org/x/sys/unix"
)

const ModelName = "RevolutionPi"

type revolutionPiBoard struct {
	resource.Named
	mu            sync.RWMutex
	logger        golog.Logger
	SPIs          []string
	I2Cs          []string
	AnalogReaders []string
	GPIONames     []string

	cancelCtx               context.Context
	cancelFunc              func()
	activeBackgroundWorkers sync.WaitGroup
}

// RegisterBoard registers a sysfs based board of the given model.
func init() {
	resource.RegisterComponent(
		board.API,
		resource.DefaultModelFamily.WithModel(ModelName),
		resource.Registration[board.Board, *Config]{Constructor: newBoard})
}

func newBoard(
	ctx context.Context,
	_ resource.Dependencies,
	conf resource.Config,
	logger golog.Logger,
) (board.Board, error) {
	cancelCtx, cancelFunc := context.WithCancel(context.Background())
	b := revolutionPiBoard{
		Named:         conf.ResourceName().AsNamed(),
		logger:        logger,
		cancelCtx:     cancelCtx,
		cancelFunc:    cancelFunc,
		SPIs:          []string{},
		I2Cs:          []string{},
		AnalogReaders: []string{},
		GPIONames:     []string{},
	}

	if err := b.Reconfigure(ctx, nil, conf); err != nil {
		return nil, err
	}
	return &b, nil
}

func (b *revolutionPiBoard) Reconfigure(
	ctx context.Context,
	_ resource.Dependencies,
	conf resource.Config,
) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	_, err := resource.NativeConfig[*Config](conf)
	if err != nil {
		return err
	}
	return nil
}

func (b *revolutionPiBoard) SPIByName(name string) (board.SPI, bool) {
	return nil, false
}

func (b *revolutionPiBoard) I2CByName(name string) (board.I2C, bool) {
	return nil, false
}

func (b *revolutionPiBoard) AnalogReaderByName(name string) (board.AnalogReader, bool) {
	return nil, false
}

func (b *revolutionPiBoard) DigitalInterruptByName(name string) (board.DigitalInterrupt, bool) {
	return nil, false // Digital interrupts aren't supported.
}

func (b *revolutionPiBoard) SPINames() []string {
	return b.SPIs
}

func (b *revolutionPiBoard) I2CNames() []string {
	return b.I2Cs
}

func (b *revolutionPiBoard) AnalogReaderNames() []string {
	return b.AnalogReaders
}

func (b *revolutionPiBoard) DigitalInterruptNames() []string {
	return nil
}

func (b *revolutionPiBoard) GPIOPinNames() []string {
	return b.GPIONames
}

func (b *revolutionPiBoard) GPIOPinByName(pinName string) (board.GPIOPin, error) {
	devPath := filepath.Join("/dev", "piControl0")
	fd, err := unix.Open(devPath, unix.O_RDONLY, 0)
	if err != nil {
		err = fmt.Errorf("open chip %v failed: %w", devPath, err)
		return nil, err
	}

	return &revolutionPiGpioPin{}, nil
}

func ioCtl(handle int, request uintptr, a uintptr) error {
	_, _, err := unix.Syscall(unix.SYS_IOCTL, uintptr(handle), request, a)
	if err != 0 {
		return err
	}
	return nil
}

func (b *revolutionPiBoard) Status(ctx context.Context, extra map[string]interface{}) (*commonpb.BoardStatus, error) {
	return &commonpb.BoardStatus{}, nil
}

func (b *revolutionPiBoard) ModelAttributes() board.ModelAttributes {
	return board.ModelAttributes{}
}

func (b *revolutionPiBoard) SetPowerMode(ctx context.Context, mode pb.PowerMode, duration *time.Duration) error {
	return grpc.UnimplementedError
}

func (b *revolutionPiBoard) Close(ctx context.Context) error {
	b.mu.Lock()
	b.cancelFunc()
	b.mu.Unlock()
	b.activeBackgroundWorkers.Wait()
	return nil
}
