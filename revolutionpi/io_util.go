//go:build linux

// Package revolutionpi implements the Revolution Pi.
package revolutionpi

func ioctlAddress(v int) int {
	kbIocMagic := int('K')
	magic := (((0) << (((0 + 8) + 8) + 14)) | ((kbIocMagic) << (0 + 8)) | ((v) << 0) | ((0) << ((0 + 8) + 8)))
	return magic
}

// The hex output received when a board is not connected.
const piControlNotConnected = 0x8000

// leaving these variables in as potential options to interface with the Revolution Pi
//
//nolint:unused
var (
	kbCmd1                 = ioctlAddress(10) // for test only
	kbCmd2                 = ioctlAddress(11) // for test only
	kbReset                = ioctlAddress(12) // reset the piControl driver including the config file
	kbGetDeviceInfoList    = ioctlAddress(13) // get the device info of all detected devices
	kbGetDeviceInfo        = ioctlAddress(14) // get the device info of one device
	kbGetValue             = ioctlAddress(15) // get the value of one bit in the process image
	kbSetValue             = ioctlAddress(16) // set the value of one bit in the process image
	kbFindVariable         = ioctlAddress(17) // find a variable defined in piCtory
	kbSetExportedOutputs   = ioctlAddress(18) // copy the exported outputs from a application process image to the real process image
	kbUpdateDeviceFirmware = ioctlAddress(19) // try to update the firmware of connected devices
	kbDIOResetCounter      = ioctlAddress(20) // set a counter or encoder to 0
	kbGetLastMessage       = ioctlAddress(21) // copy the last error message
	kbStopIO               = ioctlAddress(22) // stop/start IO communication, can be used for I/O simulation
	kbConfigStop           = ioctlAddress(23) // for download of configuration to Master Gateway: stop IO communication completely
	kbConfigSend           = ioctlAddress(24) // for download of configuration to Master Gateway: download config data
	kbConfigStart          = ioctlAddress(25) // for download of configuration to Master Gateway: restart IO communication
	// activate a watchdog for this handle. If write is not called for a given period all outputs are set to 0.
	kbSetOutputWatchdog = ioctlAddress(26)
	kbSetPos            = ioctlAddress(27) // set the f_pos, the unsigned int * is used to interpret the pos value
	kbAIOCalibrate      = ioctlAddress(28)
	kbWaitForEvent      = ioctlAddress(50) // wait for an event. This call is normally blocking
)

// SPIVariable is the struct representing an address when reading from the board.
// use kbFindVariable with ioctl to populate the struct.
type SPIVariable struct {
	strVarName  [32]byte // Variable name
	i16uAddress uint16   // Address of the byte in the process image
	i8uBit      uint8    // 0-7 bit position, >= 8 whole byte
	i16uLength  uint16   // length of the variable in bits. Possible values are 1, 8, 16 and 32
}

// SPIValue is the struct representing a value at an address when reading from the board.
// use kbGetValue or kbSetValue with ioctl to read from or write to a bit.
type SPIValue struct {
	i16uAddress uint16 // Address of the byte in the process image
	i8uBit      uint8  // 0-7 bit position, >= 8 whole byte
	i8uValue    uint8  // Value: 0/1 for bit access, whole byte otherwise
}

// SDeviceInfo is a struct representing the devices being used by the Revolution Pi module.
// use kbGetDeviceInfoList with ioctl to populate a list of these
//
//nolint:unused
type SDeviceInfo struct {
	i8uAddress       uint8     // Address of module in current configuration
	i32uSerialnumber uint32    // serial number of module
	i16uModuleType   uint16    // Type identifier of module
	i16uHWRevision   uint16    // hardware revision
	i16uSWMajor      uint16    // major software version
	i16uSWMinor      uint16    // minor software version
	i32uSVNRevision  uint32    // svn revision of software
	i16uInputLength  uint16    // length in bytes of all input values together
	i16uOutputLength uint16    // length in bytes of all output values together
	i16uConfigLength uint16    // length in bytes of all config values together
	i16uBaseOffset   uint16    // offset in process image
	i16uInputOffset  uint16    // offset in process image of first input byte
	i16uOutputOffset uint16    // offset in process image of first output byte
	i16uConfigOffset uint16    // offset in process image of first config byte
	i16uFirstEntry   uint16    // index of entry
	i16uEntries      uint16    // number of entries in process image
	i8uModuleState   uint8     // fieldbus state of piGate Module
	i8uActive        uint8     // == 0 means that the module is not present and no data is available
	i8uReserve       [30]uint8 // space for future extensions without changing the size of the struct
}

// isDIO checks whether the module is a DIO, DO, or DI module, which can be used with our GPIO related apis.
func (dev *SDeviceInfo) isDIO() bool {
	return dev.i16uModuleType == 96 || dev.i16uModuleType == 97 || dev.i16uModuleType == 98
}

// isAIO checks whether the module is an AIO module, which can be used with our Analog related apis.
func (dev *SDeviceInfo) isAIO() bool {
	return dev.i16uModuleType == 103
}

// isMIO checks whether the module is an MIO module, which can be used with the board component's Analog and GPIO related apis.
func (dev *SDeviceInfo) isMIO() bool {
	return dev.i16uModuleType == 118
}

// getModuleName gets the module name based on the module type.
func getModuleName(moduleType uint16) string {
	switch {
	case moduleType == 95:
		return "RevPi Core"
	case moduleType == 96:
		return "RevPi DIO"
	case moduleType == 97:
		return "RevPi DI"
	case moduleType == 98:
		return "RevPi DO"
	case moduleType == 103:
		return "RevPi AIO"
	case moduleType == 118:
		return "RevPi MIO"
	case moduleType == 136:
		return "RevPi Connect 4"
	case moduleType == 0x6001:
		return "ModbusTCP Slave Adapter"
	case moduleType == 0x6002:
		return "ModbusRTU Slave Adapter"
	case moduleType == 0x6003:
		return "ModbusTCP Master Adapter"
	case moduleType == 0x6004:
		return "ModbusRTU Master Adapter"
	case moduleType == 100:
		return "Gateway DMX"
	case moduleType == 71:
		return "Gateway CANopen"
	case moduleType == 73:
		return "Gateway DeviceNet"
	case moduleType == 74:
		return "Gateway EtherCAT"
	case moduleType == 75:
		return "Gateway EtherNet/IP"
	case moduleType == 93:
		return "Gateway ModbusTCP"
	case moduleType == 76:
		return "Gateway Powerlink"
	case moduleType == 77:
		return "Gateway Profibus"
	case moduleType == 79:
		return "Gateway Profinet IRT"
	case moduleType == 81:
		return "Gateway SercosIII"
	default:
		return "unknown moduletype"
	}
}
