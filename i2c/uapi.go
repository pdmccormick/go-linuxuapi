package i2c

const (
	_I2C_SLAVE       = 0x0703
	_I2C_SLAVE_FORCE = 0x0706
	_I2C_TENBIT      = 0x0704
	_I2C_FUNCS       = 0x0705
	_I2C_RDWR        = 0x0707
	_I2C_PEC         = 0x0708
	_I2C_SMBUS       = 0x0720

	_I2C_RDWR_IOCTL_MAX_MSGS = 42
)

type i2c_rdwr_ioctl_data struct {
	msgsPtr uintptr
	nmsgs   uint32
}

type i2c_msg struct {
	addr   uint16
	flags  uint16
	len    uint16
	bufPtr uintptr
}
