package efivarfs

import (
	"bytes"
	"cmp"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

var ByteOrder = binary.LittleEndian

const DefaultMountPath = `/sys/firmware/efi/efivars`

const DefaultFSPath = `/sys/firmware/efi/efivars`

var DefaultFS = FSFromMount(DefaultMountPath)

func FSFromMount(mount string) *FS {
	return &FS{
		FS:    os.DirFS(mount),
		Mount: mount,
	}
}

// Support for Linux efivarfs
type FS struct {
	FS    fs.FS
	Mount string
}

var (
	_ fs.FS     = (*FS)(nil)
	_ fs.GlobFS = (*FS)(nil)
)

func (fsys *FS) Open(name string) (fs.File, error)     { return fsys.FS.Open(name) }
func (fsys *FS) Glob(pattern string) ([]string, error) { return fs.Glob(fsys.FS, pattern) }

func (fsys *FS) GlobNames(pattern, guid string) (names Names, err error) {
	dirents, err := fs.ReadDir(fsys.FS, ".")
	if err != nil {
		return nil, err
	}

	for _, dirent := range dirents {
		if dirent.IsDir() {
			continue
		}

		var (
			filename = filepath.Base(dirent.Name())
			name, ok = NameFromFilename(filename)
		)

		if !ok {
			continue
		}

		if guid != "" && name.GUID != guid {
			continue
		}

		if pattern != "" {
			matched, err := path.Match(pattern, name.Id)
			if err != nil {
				return nil, err
			}

			if !matched {
				continue
			}
		}

		names = append(names, name)
	}

	names.SortByGUID()

	return names, nil
}

func (fsys *FS) ListNames() (names Names, err error) {
	return fsys.GlobNames("", "")
}

func (fsys *FS) GetAll() (vars []Variable, err error) {
	names, err := fsys.ListNames()
	if err != nil {
		return nil, err
	}

	var errs []error

	for _, name := range names {
		var v = Variable{
			Name: name,
		}

		if err := fsys.Get(&v); err != nil {
			errs = append(errs, err)
		} else {
			vars = append(vars, v)
		}
	}

	if len(errs) > 0 {
		err = errors.Join(errs...)
	}

	return
}

func (fsys *FS) Get(v *Variable) error {
	val, err := fsys.GetValue(v.Name)
	if err != nil {
		return err
	}

	v.Value = *val
	return nil
}

func (fsys *FS) Set(v *Variable) error { return fsys.SetValue(v.Name, &v.Value) }

func (fsys *FS) GetValue(name Name) (*Value, error) {
	f, err := fsys.FS.Open(name.Filename())
	if err != nil {
		return nil, err
	}

	defer f.Close()

	var val Value
	if _, err := val.ReadFrom(f); err != nil {
		return nil, err
	}

	return &val, nil
}

var ErrNoMount = errors.New("efivarfs: FS is missing Mount for writing")

func (fsys *FS) writeValue(path string, flags fsFlags, val *Value) error {
	fw, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}

	defer func() {
		if !flags.Immutable() {
			flags.SetImmutable()
			flags.IoctlSet(fw)
		}

		fw.Close()
	}()

	if _, err := val.WriteTo(fw); err != nil {
		return err
	}

	return nil
}

func (fsys *FS) SetValue(name Name, val *Value) error {
	if fsys.Mount == "" {
		return ErrNoMount
	}

	var flags fsFlags

	path := filepath.Join(fsys.Mount, name.Filename())
	f, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	} else {
		defer f.Close()

		if err := flags.IoctlGet(f); err != nil {
			return err
		}

		if flags.Immutable() {
			flags.ClearImmutable()
			if err := flags.IoctlSet(f); err != nil {
				return err
			}
		}
	}

	return fsys.writeValue(path, flags, val)
}

func (fsys *FS) RemoveName(name Name) error {
	if fsys.Mount == "" {
		return ErrNoMount
	}

	path := filepath.Join(fsys.Mount, name.Filename())
	f, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return err
	}

	defer func() {
		if f != nil {
			f.Close()
		}
	}()

	var flags fsFlags
	if err := flags.IoctlGet(f); err != nil {
		return err
	}

	if flags.Immutable() {
		flags.ClearImmutable()
		if err := flags.IoctlSet(f); err != nil {
			return err
		}
	}

	f.Close()
	f = nil

	return os.Remove(path)
}

// EFI variable attributes. Defined in [Runtime Services] from the UEFI specification.
//
// [Runtime Services]: https://uefi.org/specs/UEFI/2.10_A/08_Services_Runtime_Services.html#getvariable
type Attrs uint32

const (
	NonVolatile              Attrs = 0x00000001 // EFI_VARIABLE_NON_VOLATILE
	BootServiceAccess        Attrs = 0x00000002 // EFI_VARIABLE_BOOTSERVICE_ACCESS
	RuntimeAccess            Attrs = 0x00000004 // EFI_VARIABLE_RUNTIME_ACCESS
	HardwareRecordError      Attrs = 0x00000008 // EFI_VARIABLE_HARDWARE_ERROR_RECORD
	TimeBasedAuthWriteAccess Attrs = 0x00000020 // EFI_VARIABLE_TIME_BASED_AUTHENTICATED_WRITE_ACCESS
	AppendWrite              Attrs = 0x00000040 // EFI_VARIABLE_APPEND_WRITE

	// EFI_VARIABLE_ENHANCED_AUTHENTICATED_ACCESS. Variable payload begins with an [EFI_VARIABLE_AUTHENTICATION_3] structure.
	//
	// [EFI_VARIABLE_AUTHENTICATION_3]: https://uefi.org/specs/UEFI/2.10_A/08_Services_Runtime_Services.html#using-the-efi-variable-authentication-3-descriptor
	EnhancedAuthAccess Attrs = 0x00000080
)

var attrBits = [...]struct {
	Mask Attrs
	Name string
}{
	{NonVolatile, "NV"},
	{BootServiceAccess, "BS"},
	{RuntimeAccess, "RA"},
	{HardwareRecordError, "HR"},
	{TimeBasedAuthWriteAccess, "TB"},
	{AppendWrite, "AP"},
	{EnhancedAuthAccess, "EA"},
}

func (v Attrs) String() string {
	var (
		slots [len(attrBits) + 1]string
		out   = slots[:0]
	)

	for _, def := range attrBits {
		if m := def.Mask; (v & m) == m {
			v &^= m
			out = append(out, def.Name)
		}
	}

	if v != 0 {
		out = append(out, fmt.Sprintf("0x%x", uint32(v)))
	}

	return strings.Join(out, " ")
}

type Name struct {
	GUID string
	Id   string
}

const GUIDFormatPattern = `(([[:xdigit:]]{8})-([[:xdigit:]]{4})-([[:xdigit:]]{4})-([[:xdigit:]]{4})-([[:xdigit:]]{12}))`

var FilenameMatcher = regexp.MustCompile(`(.+)-` + GUIDFormatPattern)

func NameFromFilename(filename string) (name Name, ok bool) {
	ms := FilenameMatcher.FindAllStringSubmatch(filename, 1)
	if len(ms) < 1 {
		return
	}

	m := ms[0]
	name = Name{
		GUID: m[2],
		Id:   m[1],
	}
	return name, true
}

func (n Name) String() string { return n.GUID + " " + n.Id }

func (n *Name) Filename() string { return n.Id + "-" + n.GUID }

func (a *Name) compareTo(b *Name) int {
	if c := a.compareTo_GUID(b); c != 0 {
		return c
	} else {
		return a.compareTo_Id(b)
	}
}

func (a *Name) compareToFlip(b *Name) int {
	if c := a.compareTo_Id(b); c != 0 {
		return c
	} else {
		return a.compareTo_GUID(b)
	}
}

func (a *Name) compareTo_GUID(b *Name) int {
	return cmp.Compare(strings.ToLower(a.GUID), strings.ToLower(b.GUID))
}
func (a *Name) compareTo_Id(b *Name) int {
	return cmp.Compare(strings.ToLower(a.Id), strings.ToLower(b.Id))
}

type Names []Name

func (names *Names) Append(elems ...Name) Names {
	*names = append(*names, elems...)
	return *names
}

func nameGuidCompare(a, b Name) int { return a.compareTo(&b) }
func nameIdCompare(a, b Name) int   { return a.compareToFlip(&b) }

// Sort names by GUID, then Id.
func (names Names) SortByGUID() { slices.SortFunc(names, nameGuidCompare) }

// Sort names by Id, then GUID.
func (names Names) SortById() { slices.SortFunc(names, nameIdCompare) }

type Value struct {
	Attrs Attrs
	Data  []byte
}

var (
	_ io.WriterTo   = (*Value)(nil)
	_ io.ReaderFrom = (*Value)(nil)
)

func (v *Value) Size() int { return 4 + len(v.Data) }

var ErrBufferTooShort = errors.New("efivarfs: buffer too short")

func (v *Value) WriteTo(w io.Writer) (n int64, err error) {
	buf := make([]byte, v.Size())
	if _, err := v.Encode(buf); err != nil {
		return 0, err
	}

	n0, err := w.Write(buf)
	n += int64(n0)
	if err != nil {
		return n, fmt.Errorf("write1: %w", err)
		return n, err
	}

	return n, nil
}

func (v *Value) ReadFrom(r io.Reader) (n int64, err error) {
	var attrs [4]byte
	n0, err := io.ReadFull(r, attrs[:])
	n += int64(n0)
	if err != nil {
		return n, err
	}

	data, err := io.ReadAll(r)
	n += int64(len(data))
	if err != nil {
		return n, err
	}

	v.Attrs = Attrs(ByteOrder.Uint32(attrs[:]))
	v.Data = data
	return n, nil
}

func (v *Value) Encode(buf []byte) (n int, err error) {
	n = v.Size()
	if len(buf) < n {
		return 0, ErrBufferTooShort
	}

	ByteOrder.PutUint32(buf, uint32(v.Attrs))
	copy(buf[4:], v.Data)
	return
}

func (v *Value) Decode(buf []byte) (n int, err error) {
	if len(buf) < 4 {
		return 0, ErrBufferTooShort
	}

	v.Attrs = Attrs(ByteOrder.Uint32(buf))
	if tail := buf[4:]; len(tail) > 0 {
		v.Data = bytes.Clone(tail)
	}

	return len(buf), nil
}

type Variable struct {
	Name
	Value
}

func (v *Variable) GetValue(fsys *FS) error { return fsys.Get(v) }
func (v *Variable) SetValue(fsys *FS) error { return fsys.Set(v) }
