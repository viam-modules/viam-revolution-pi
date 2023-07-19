//go:build linux

package revolution_pi

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"

	"github.com/edaniels/golog"
	"golang.org/x/sys/unix"
)

type gpioChip struct {
	dev        string
	logger     golog.Logger
	fileHandle *os.File
}

func (g *gpioChip) GetGPIOPin(pinName string) (*gpioPin, error) {
	pin := SPIVariable{strVarName: Char32(pinName)}
	err := g.mapNameToAddress(&pin)
	if err != nil {
		return nil, err
	}
	g.logger.Debugf("Found pin address: %#v", pin)
	gpioPin := gpioPin{Name: Str32(pin.strVarName), Address: pin.i16uAddress, BitPosition: pin.i8uBit, Length: pin.i16uLength, ControlChip: g}
	gpioPin.initialize()
	return &gpioPin, nil
}

func (g *gpioChip) mapNameToAddress(pin *SPIVariable) error {
	g.logger.Debugf("Looking for address of %#v", pin)
	err := g.ioCtl(uintptr(KB_FIND_VARIABLE), unsafe.Pointer(pin))
	if err != 0 {
		e := fmt.Errorf("failed to get pin address info %v failed: %w", g.dev, err)
		return e
	}
	return nil
}

func (b *gpioChip) GetAnalogInput(pinName string) (*analogPin, error) {
	pin := SPIVariable{strVarName: Char32(pinName)}
	err := b.mapNameToAddress(&pin)
	if err != nil {
		return nil, err
	}
	b.logger.Infof("Found pin address: %#v", pin)
	return &analogPin{Name: Str32(pin.strVarName), Address: pin.i16uAddress, Length: pin.i16uLength}, nil
}

func (g *gpioChip) ioCtl(command uintptr, message unsafe.Pointer) syscall.Errno {
	handle := g.fileHandle.Fd()
	g.logger.Debugf("Handle: %#v, Command: %#v, Message: %#V", handle, command, message)
	_, _, err := unix.Syscall(unix.SYS_IOCTL, handle, command, uintptr(message))
	return err
}

func (g *gpioChip) getBitValue(address int64, bitPosition uint8) (bool, error) {
	b := make([]byte, 1)
	n, err := g.fileHandle.ReadAt(b, address)
	g.logger.Debugf("Read %#v bytes", b)
	if n != 1 {
		return false, fmt.Errorf("expected 1 byte, got %#v", b)
	}
	if err != nil {
		return false, err
	}
	if (b[0]>>bitPosition)&1 == 1 {
		return true, nil
	} else {
		return false, nil
	}
}

func (g *gpioChip) writeValue(address int64, b []byte) error {
	g.logger.Infof("Writing %#v to %v", b, address)
	n, err := g.fileHandle.WriteAt(b, address)
	g.logger.Infof("Wrote %#v byte(s), n: %v", b, n)
	if err != nil {
		return err
	}
	if n < 1 || n > 1 {
		return fmt.Errorf("expected 1 byte(s), got %#v", b)
	}
	return nil
}

func (g *gpioChip) Close() error {
	err := g.fileHandle.Close()
	return err
}
