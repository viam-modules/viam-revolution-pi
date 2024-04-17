//go:build linux

// Package revolutionpi implements the Revolution Pi.
package revolutionpi

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"

	"go.uber.org/multierr"
	"go.viam.com/rdk/logging"
	"golang.org/x/sys/unix"
)

type gpioChip struct {
	dev        string
	logger     logging.Logger
	fileHandle *os.File
	dioDevices []SDeviceInfo
}

func (g *gpioChip) GetGPIOPin(pinName string) (*gpioPin, error) {
	pin := SPIVariable{strVarName: char32(pinName)}
	err := g.mapNameToAddress(&pin)
	if err != nil {
		return nil, err
	}
	g.logger.Debugf("Found GPIO pin: %#v", pin)
	gpioPin := gpioPin{Name: str32(pin.strVarName), Address: pin.i16uAddress, BitPosition: pin.i8uBit, Length: pin.i16uLength, ControlChip: g}
	err = gpioPin.initialize()
	if err != nil {
		return nil, err
	}
	return &gpioPin, nil
}

func (g *gpioChip) GetAnalogInput(pinName string) (*analogPin, error) {
	pin := SPIVariable{strVarName: char32(pinName)}
	err := g.mapNameToAddress(&pin)
	if err != nil {
		return nil, err
	}
	g.logger.Debugf("Found Analog pin: %#v", pin)
	return &analogPin{Name: str32(pin.strVarName), Address: pin.i16uAddress, Length: pin.i16uLength, ControlChip: g}, nil
}

func (g *gpioChip) mapNameToAddress(pin *SPIVariable) error {
	g.logger.Debugf("Looking for address of %#v", pin)
	//nolint:gosec
	err := g.ioCtl(uintptr(kbFindVariable), unsafe.Pointer(pin))
	if err != 0 {
		e := fmt.Errorf("failed to get pin address info %v failed: %w", g.dev, err)
		return e
	}
	g.logger.Debugf("Found address of %#v", pin)
	return nil
}

func (g *gpioChip) showDeviceList() error {
	var deviceInfoList [255]SDeviceInfo
	g.dioDevices = []SDeviceInfo{}
	//nolint:gosec
	cnt, _, err := g.ioCtlReturns(uintptr(kbGetDeviceInfoList), unsafe.Pointer(&deviceInfoList))
	if err != 0 {
		e := fmt.Errorf("failed to retrieve device info list: %d", -int(cnt))
		return e
	}

	var deviceErrs error
	for i := 0; i < int(cnt); i++ {
		if deviceInfoList[i].i8uActive != 0 {
			g.logger.Debugf("device %d is of type %s is active", i, getModuleName(deviceInfoList[i].i16uModuleType))
			if deviceInfoList[i].isDIO() {
				g.logger.Debugf("device info: %v", deviceInfoList[i])
				g.dioDevices = append(g.dioDevices, deviceInfoList[i])
			}
		} else {
			checkConnected := deviceInfoList[i].i16uModuleType&piControlNotConnected == piControlNotConnected
			if checkConnected {
				deviceErr := fmt.Errorf("device %d is not connected", i)
				deviceErrs = multierr.Combine(deviceErrs, deviceErr)
			} else {
				deviceErr := fmt.Errorf("device %d is type %s but is not configured", i, getModuleName(deviceInfoList[i].i16uModuleType))
				deviceErrs = multierr.Combine(deviceErrs, deviceErr)
			}
		}
	}
	return deviceErrs
}

func (g *gpioChip) ioCtl(command uintptr, message unsafe.Pointer) syscall.Errno {
	_, _, err := g.ioCtlReturns(command, message)
	return err
}

func (g *gpioChip) ioCtlReturns(command uintptr, message unsafe.Pointer) (uintptr, uintptr, syscall.Errno) {
	handle := g.fileHandle.Fd()
	g.logger.Debugf("Handle: %#v, Command: %#v, Message: %#v", handle, command, message)
	return unix.Syscall(unix.SYS_IOCTL, handle, command, uintptr(message))
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
	}
	return false, nil
}

func (g *gpioChip) writeValue(address int64, b []byte) error {
	g.logger.Debugf("Writing %#d to %v", b, address)
	n, err := g.fileHandle.WriteAt(b, address)
	g.logger.Debugf("Wrote %#d byte(s), n: %d", b, n)
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

func (g *gpioChip) findDIODevice(gpioAddress uint16) (SDeviceInfo, error) {
	for _, dev := range g.dioDevices {
		// need to test devOffsetLower with multiple DIO devices
		devOffsetLower := dev.i16uInputOffset
		devOffsetUpper := dev.i16uInputOffset + dev.i16uOutputLength + dev.i16uInputLength + dev.i16uConfigLength
		if gpioAddress >= devOffsetLower && gpioAddress < devOffsetUpper {
			return dev, nil
		}
	}
	return SDeviceInfo{}, fmt.Errorf("unable to find device for pin %d", gpioAddress)
}
