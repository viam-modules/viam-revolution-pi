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

func (b *gpioChip) GetBinaryIOPin(pinName string) (*gpioPin, error) {
	pin := SPIVariable{strVarName: Char32(pinName)}
	err := b.mapNameToAddress(&pin)
	if err != nil {
		return nil, err
	}
	b.logger.Debugf("Found pin address: %#v", pin)
	return &gpioPin{Name: Str32(pin.strVarName), Address: pin.i16uAddress, BitPosition: pin.i8uBit, Length: pin.i16uLength, ControlChip: b}, nil
}

func (b *gpioChip) mapNameToAddress(pin *SPIVariable) error {
	b.logger.Debugf("Looking for address of %#v", pin)
	err := b.ioCtl(uintptr(KB_FIND_VARIABLE), unsafe.Pointer(pin))
	if err != 0 {
		e := fmt.Errorf("failed to get pin address info %v failed: %w", b.dev, err)
		return e
	}
	return nil
}

func (b *gpioChip) ioCtl(command uintptr, message unsafe.Pointer) syscall.Errno {
	handle := b.fileHandle.Fd()
	b.logger.Debugf("Handle: %#v, Command: %#v, Message: %#V", handle, command, message)
	_, _, err := unix.Syscall(unix.SYS_IOCTL, handle, command, uintptr(message))
	return err
}
