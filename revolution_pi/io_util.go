package revolution_pi

func ioctl_address(v int) int {
	kb_ioc_magic := int('K')
	magic := (((0) << (((0 + 8) + 8) + 14)) | ((kb_ioc_magic) << (0 + 8)) | ((v) << 0) | ((0) << ((0 + 8) + 8)))
	return magic
}

var KB_CMD1 = ioctl_address(10)                   // for test only
var KB_CMD2 = ioctl_address(11)                   // for test only
var KB_RESET = ioctl_address(12)                  // reset the piControl driver including the config file
var KB_GET_DEVICE_INFO_LIST = ioctl_address(13)   // get the device info of all detected devices
var KB_GET_DEVICE_INFO = ioctl_address(14)        // get the device info of one device
var KB_GET_VALUE = ioctl_address(15)              // get the value of one bit in the process image
var KB_SET_VALUE = ioctl_address(16)              // set the value of one bit in the process image
var KB_FIND_VARIABLE = ioctl_address(17)          // find a varible defined in piCtory
var KB_SET_EXPORTED_OUTPUTS = ioctl_address(18)   // copy the exported outputs from a application process image to the real process image
var KB_UPDATE_DEVICE_FIRMWARE = ioctl_address(19) // try to update the firmware of connected devices
var KB_DIO_RESET_COUNTER = ioctl_address(20)      // set a counter or endocder to 0
var KB_GET_LAST_MESSAGE = ioctl_address(21)       // copy the last error message
var KB_STOP_IO = ioctl_address(22)                // stop/start IO communication, can be used for I/O simulation
var KB_CONFIG_STOP = ioctl_address(23)            // for download of configuration to Master Gateway: stop IO communication completely
var KB_CONFIG_SEND = ioctl_address(24)            // for download of configuration to Master Gateway: download config data
var KB_CONFIG_START = ioctl_address(25)           // for download of configuration to Master Gateway: restart IO communication
var KB_SET_OUTPUT_WATCHDOG = ioctl_address(26)    // activate a watchdog for this handle. If write is not called for a given period all outputs are set to 0
var KB_SET_POS = ioctl_address(27)                // set the f_pos, the unsigned int * is used to interpret the pos value
var KB_AIO_CALIBRATE = ioctl_address(28)

var KB_WAIT_FOR_EVENT = ioctl_address(50) // wait for an event. This call is normally blocking

type SPIVariable struct {
	strVarName  [32]byte // Variable name
	i16uAddress uint16   // Address of the byte in the process image
	i8uBit      uint8    // 0-7 bit position, >= 8 whole byte
	i16uLength  uint16   // length of the variable in bits. Possible values are 1, 8, 16 and 32
}

type SPIValue struct {
	i16uAddress uint16 // Address of the byte in the process image
	i8uBit      uint8  // 0-7 bit position, >= 8 whole byte
	i8uValue    uint8  // Value: 0/1 for bit access, whole byte otherwise
}

type PwmStateRequest struct {
	i16uAddress uint16 // Address of the byte in the process image
	i8uBit      uint8  // 0-7 bit position, >= 8 whole byte
	i8uValue    uint16 // Value: 0/1 for bit access, whole byte otherwise
}
