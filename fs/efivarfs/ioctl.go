package efivarfs

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

// From <uapi/linux/fs.h>
const (
	_FS_IMMUTABLE_FL = 0x00000010
)

type fsFlags int

func (flags fsFlags) Immutable() bool  { return (flags & _FS_IMMUTABLE_FL) != 0 }
func (flags *fsFlags) SetImmutable()   { *flags |= _FS_IMMUTABLE_FL }
func (flags *fsFlags) ClearImmutable() { *flags &^= _FS_IMMUTABLE_FL }
func (flags *fsFlags) UpdateImmutable(set bool) {
	if set {
		flags.SetImmutable()
	} else {
		flags.ClearImmutable()
	}
}

func (flags *fsFlags) IoctlGet(f *os.File) error {
	attr, err := unix.IoctlGetInt(int(f.Fd()), unix.FS_IOC_GETFLAGS)
	if err != nil {
		return fmt.Errorf("ioctl FS_IOC_GETFLAGS: %w", err)
	}
	*flags = (fsFlags)(attr)
	return nil
}

func (flags fsFlags) IoctlSet(f *os.File) error {
	return unix.IoctlSetPointerInt(int(f.Fd()), unix.FS_IOC_SETFLAGS, int(flags))
}
