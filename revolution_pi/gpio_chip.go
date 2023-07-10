//go:build linux

package revolution_pi

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/unix"
)

type gpioChip struct {
	handle int
	dev    string
}

func (b *gpioChip) GetBinaryIOPin(pinName string) (*gpioPin, error) {
	pin := SPIVariable{strVarName: Char32(pinName)}
	err := b.mapNameToAddress(&pin)
	if err != nil {
		return nil, err
	}
	return &gpioPin{Name: Str32(pin.strVarName), Address: pin.i16uAddress, BitPosition: pin.i8uBit, Length: pin.i16uLength}, nil
}

func (b *gpioChip) mapNameToAddress(pin *SPIVariable) error {
	err := b.ioCtl(uintptr(KB_FIND_VARIABLE), uintptr(unsafe.Pointer(&pin)))
	if err != nil {
		e := fmt.Errorf("failed to get pin address info %v failed: %w", b.dev, err)
		return e
	}
	return nil
}

func (b *gpioChip) ioCtl(command uintptr, message uintptr) error {
	_, _, err := unix.Syscall(unix.SYS_IOCTL, uintptr(b.handle), command, message)
	return err
}
