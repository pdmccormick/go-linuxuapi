package i2c

import (
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

// See <https://docs.kernel.org/i2c/dev-interface.html>

// Device
type Device struct {
	f     *os.File
	Funcs uintptr
}

func OpenDevice(name string) (dev *Device, err error) {
	var f *os.File

	defer func() {
		if err != nil && f != nil {
			f.Close()
		}
	}()

	f, err = os.OpenFile(name, os.O_RDWR, 0)
	if err != nil {
		return
	}

	dev = &Device{
		f: f,
	}

	if v, err := dev.getFuncs(); err != nil {
		return nil, err
	} else {
		dev.Funcs = v
	}

	return
}

func (dev *Device) Close() error { return dev.f.Close() }

func (dev *Device) ioctl(mode, arg uintptr) error {
	if _, _, errno := unix.Syscall(unix.SYS_IOCTL, dev.f.Fd(), mode, arg); errno != 0 {
		return errno
	}
	return nil
}

func (dev *Device) getFuncs() (v uintptr, err error) {
	err = dev.ioctl(_I2C_FUNCS, uintptr(unsafe.Pointer(&v)))
	return
}

func (dev *Device) Rdwr(msgs []Msg) error {
	var (
		raw   [_I2C_RDWR_IOCTL_MAX_MSGS]i2c_msg
		cmsgs = raw[:min(len(msgs), len(raw))]
		req   = i2c_rdwr_ioctl_data{
			msgsPtr: uintptr(unsafe.Pointer(&cmsgs[0])),
			nmsgs:   uint32(len(cmsgs)),
		}
	)

	defer func() {
		// Expunge all trace of pointer values
		req.msgsPtr = 0

		for i := range cmsgs {
			cmsgs[i].bufPtr = 0
		}
	}()

	for i := range cmsgs {
		cmsgs[i] = msgs[i].toC()
	}

	return dev.ioctl(_I2C_RDWR, uintptr(unsafe.Pointer(&req)))
}

func (dev *Device) ReadReg(addr uint16, reg byte) (byte, error) {
	var (
		outbuf = [1]byte{reg}
		inbuf  [1]byte
		msgs   = [2]Msg{
			{Addr: addr, Flags: 0, Buf: outbuf[:]},
			{Addr: addr, Flags: MsgRead | MsgNoStart, Buf: inbuf[:]},
		}
	)

	if err := dev.Rdwr(msgs[:]); err != nil {
		return 0, err
	}

	return inbuf[0], nil
}

func (dev *Device) WriteReg(addr uint16, reg, value byte) error {
	var (
		outbuf = [2]byte{reg, value}
		msgs   = [1]Msg{{Addr: addr, Flags: 0, Buf: outbuf[:]}}
	)

	return dev.Rdwr(msgs[:])
}

func (dev *Device) Txn(addr uint16, w, r []byte) error {
	if w == nil {
		var one [1]byte
		w = one[:]
	}

	var (
		raw = [2]Msg{
			{Addr: addr, Flags: 0, Buf: w},
			{Addr: addr, Flags: MsgRead | MsgNoStart, Buf: r},
		}
		msgs = raw[:2]
	)

	if r == nil {
		msgs = raw[:1]
	}

	return dev.Rdwr(msgs)
}

// Msg
type Msg struct {
	Addr  uint16
	Flags int
	Buf   []byte
}

func (msg *Msg) toC() (out i2c_msg) {
	out = i2c_msg{
		addr:  msg.Addr,
		flags: uint16(msg.Flags),
		len:   uint16(len(msg.Buf)),
	}

	if out.len > 0 {
		out.bufPtr = uintptr(unsafe.Pointer(&msg.Buf[0]))
	}

	return
}

const (
	MsgRead       = 0x0001
	MsgTen        = 0x0010
	MsgDmaSafe    = 0x0200
	MsgRecvLen    = 0x0400
	MsgNoReadAck  = 0x0800
	MsgIgnoreNak  = 0x1000
	MsgRevDirAddr = 0x2000
	MsgNoStart    = 0x4000
	MsgSstop      = 0x8000
)
