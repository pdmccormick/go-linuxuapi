package hidgadget

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Device struct {
	f                *os.File
	keyPressDuration time.Duration
}

const DefaultKeyPressDuration = 10 * time.Millisecond

func OpenDevice(name string, keyPressDuration time.Duration) (*Device, error) {
	if name == "" {
		name = "/dev/hidg0"
	}

	if !strings.HasPrefix(name, "/") && !strings.HasPrefix(name, "./") {
		name = filepath.Join("/dev", name)
	}

	f, err := os.OpenFile(name, os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}

	if keyPressDuration == 0 {
		keyPressDuration = DefaultKeyPressDuration
	}

	var dev = Device{f: f, keyPressDuration: keyPressDuration}
	return &dev, nil
}

func (dev *Device) holdPause() { time.Sleep(dev.keyPressDuration) }

func (dev *Device) Close() error { return dev.f.Close() }

func (dev *Device) SendReport(r *Report) error {
	var b = r.Raw()
	_, err := dev.f.Write(b[:])
	return err
}

func (dev *Device) HoldKeys(keys ...Key) error {
	var r Report
	r.SetKeys(keys...)
	return dev.SendReport(&r)
}

func (dev *Device) ReleaseKeys() error {
	var b [ReportLen]byte
	_, err := dev.f.Write(b[:])
	return err
}

func (dev *Device) ChordKeys(keys ...Key) error {
	if err := dev.HoldKeys(keys...); err != nil {
		return err
	}

	dev.holdPause()

	return dev.ReleaseKeys()
}

func (dev *Device) PressKeys(keys ...Key) error {
	for _, k := range keys {
		if k == 0 {
			continue
		}

		if err := dev.HoldKeys(k); err != nil {
			return err
		}

		dev.holdPause()

		if err := dev.ReleaseKeys(); err != nil {
			return err
		}

		dev.holdPause()
	}

	return nil
}

type MissingKeymapError byte

func (c MissingKeymapError) Error() string {
	return fmt.Sprintf("Missing mapping for character `%c`", c)
}

func (dev *Device) TypeText(text string, keymap Keymap) error {
	if keymap == nil {
		keymap = USQwertyKeyboard
	}

	for _, c := range []byte(text) {
		k, ok := keymap[c]
		if !ok || k == 0 {
			return MissingKeymapError(c)
		}

		if err := dev.PressKeys(k); err != nil {
			return err
		}
	}

	return nil
}

type Modifier byte

const (
	Modifier_LeftCtrl   Modifier = 0x01
	Modifier_RightCtrl  Modifier = 0x10
	Modifier_LeftShift  Modifier = 0x02
	Modifier_RightShift Modifier = 0x20
	Modifier_LeftAlt    Modifier = 0x04
	Modifier_RightAlt   Modifier = 0x40
	Modifier_LeftMeta   Modifier = 0x08
	Modifier_RightMeta  Modifier = 0x80

	Key_LeftCtrl   = Key(Modifier_LeftCtrl) << 8
	Key_RightCtrl  = Key(Modifier_RightCtrl) << 8
	Key_LeftShift  = Key(Modifier_LeftShift) << 8
	Key_RightShift = Key(Modifier_RightShift) << 8
	Key_LeftAlt    = Key(Modifier_LeftAlt) << 8
	Key_RightAlt   = Key(Modifier_RightAlt) << 8
	Key_LeftMeta   = Key(Modifier_LeftMeta) << 8
	Key_RightMeta  = Key(Modifier_RightMeta) << 8
)

func (m Modifier) Key() Key { return Key(m) << 8 }

type Key uint16

const (
	Key_None                 Key = 0x00_00
	Key_ErrorRollOver        Key = 0x00_01
	Key_POSTFail             Key = 0x00_02
	Key_ErrorUndefined       Key = 0x00_03
	Key_A                    Key = 0x00_04
	Key_B                    Key = 0x00_05
	Key_C                    Key = 0x00_06
	Key_D                    Key = 0x00_07
	Key_E                    Key = 0x00_08
	Key_F                    Key = 0x00_09
	Key_G                    Key = 0x00_0A
	Key_H                    Key = 0x00_0B
	Key_I                    Key = 0x00_0C
	Key_J                    Key = 0x00_0D
	Key_K                    Key = 0x00_0E
	Key_L                    Key = 0x00_0F
	Key_M                    Key = 0x00_10
	Key_N                    Key = 0x00_11
	Key_O                    Key = 0x00_12
	Key_P                    Key = 0x00_13
	Key_Q                    Key = 0x00_14
	Key_R                    Key = 0x00_15
	Key_S                    Key = 0x00_16
	Key_T                    Key = 0x00_17
	Key_U                    Key = 0x00_18
	Key_V                    Key = 0x00_19
	Key_W                    Key = 0x00_1A
	Key_X                    Key = 0x00_1B
	Key_Y                    Key = 0x00_1C
	Key_Z                    Key = 0x00_1D
	Key_1                    Key = 0x00_1E
	Key_2                    Key = 0x00_1F
	Key_3                    Key = 0x00_20
	Key_4                    Key = 0x00_21
	Key_5                    Key = 0x00_22
	Key_6                    Key = 0x00_23
	Key_7                    Key = 0x00_24
	Key_8                    Key = 0x00_25
	Key_9                    Key = 0x00_26
	Key_0                    Key = 0x00_27
	Key_Enter                Key = 0x00_28
	Key_Escape               Key = 0x00_29
	Key_Backspace            Key = 0x00_2A
	Key_Tab                  Key = 0x00_2B
	Key_Space                Key = 0x00_2C
	Key_Minus                Key = 0x00_2D
	Key_Equal                Key = 0x00_2E
	Key_LeftBracket          Key = 0x00_2F
	Key_RightBracket         Key = 0x00_30
	Key_Backslash            Key = 0x00_31
	Key_NonUSHash            Key = 0x00_32
	Key_Semicolon            Key = 0x00_33
	Key_Apostrophe           Key = 0x00_34
	Key_Grave                Key = 0x00_35
	Key_Comma                Key = 0x00_36
	Key_Period               Key = 0x00_37
	Key_Slash                Key = 0x00_38
	Key_CapsLock             Key = 0x00_39
	Key_F1                   Key = 0x00_3A
	Key_F2                   Key = 0x00_3B
	Key_F3                   Key = 0x00_3C
	Key_F4                   Key = 0x00_3D
	Key_F5                   Key = 0x00_3E
	Key_F6                   Key = 0x00_3F
	Key_F7                   Key = 0x00_40
	Key_F8                   Key = 0x00_41
	Key_F9                   Key = 0x00_42
	Key_F10                  Key = 0x00_43
	Key_F11                  Key = 0x00_44
	Key_F12                  Key = 0x00_45
	Key_PrintScreen          Key = 0x00_46
	Key_ScrollLock           Key = 0x00_47
	Key_Pause                Key = 0x00_48
	Key_Insert               Key = 0x00_49
	Key_Home                 Key = 0x00_4A
	Key_PageUp               Key = 0x00_4B
	Key_Delete               Key = 0x00_4C
	Key_End                  Key = 0x00_4D
	Key_PageDown             Key = 0x00_4E
	Key_RightArrow           Key = 0x00_4F
	Key_LeftArrow            Key = 0x00_50
	Key_DownArrow            Key = 0x00_51
	Key_UpArrow              Key = 0x00_52
	Key_NumLock              Key = 0x00_53
	Key_PadSlash             Key = 0x00_54
	Key_PadAsterisk          Key = 0x00_55
	Key_PadMinus             Key = 0x00_56
	Key_PadPlus              Key = 0x00_57
	Key_PadEnter             Key = 0x00_58
	Key_Pad1                 Key = 0x00_59
	Key_Pad2                 Key = 0x00_5A
	Key_Pad3                 Key = 0x00_5B
	Key_Pad4                 Key = 0x00_5C
	Key_Pad5                 Key = 0x00_5D
	Key_Pad6                 Key = 0x00_5E
	Key_Pad7                 Key = 0x00_5F
	Key_Pad8                 Key = 0x00_60
	Key_Pad9                 Key = 0x00_61
	Key_Pad0                 Key = 0x00_62
	Key_PadPeriod            Key = 0x00_63
	Key_NonUSBackslash       Key = 0x00_64
	Key_Application          Key = 0x00_65
	Key_Power                Key = 0x00_66
	Key_PadEqual             Key = 0x00_67
	Key_F13                  Key = 0x00_68
	Key_F14                  Key = 0x00_69
	Key_F15                  Key = 0x00_6A
	Key_F16                  Key = 0x00_6B
	Key_F17                  Key = 0x00_6C
	Key_F18                  Key = 0x00_6D
	Key_F19                  Key = 0x00_6E
	Key_F20                  Key = 0x00_6F
	Key_F21                  Key = 0x00_70
	Key_F22                  Key = 0x00_71
	Key_F23                  Key = 0x00_72
	Key_F24                  Key = 0x00_73
	Key_Execute              Key = 0x00_74
	Key_Help                 Key = 0x00_75
	Key_Menu                 Key = 0x00_76
	Key_Select               Key = 0x00_77
	Key_Stop                 Key = 0x00_78
	Key_Again                Key = 0x00_79
	Key_Undo                 Key = 0x00_7A
	Key_Cut                  Key = 0x00_7B
	Key_Copy                 Key = 0x00_7C
	Key_Paste                Key = 0x00_7D
	Key_Find                 Key = 0x00_7E
	Key_Mute                 Key = 0x00_7F
	Key_VolumeUp             Key = 0x00_80
	Key_VolumeDown           Key = 0x00_81
	Key_LockingCapsLock      Key = 0x00_82
	Key_LockingNumLock       Key = 0x00_83
	Key_LockingScrollLockKey     = 0x00_84
	Key_PadComma             Key = 0x00_85
	Key_PadEqualSign         Key = 0x00_86
	Key_International1       Key = 0x00_87
	Key_International2       Key = 0x00_88
	Key_International3       Key = 0x00_89
	Key_International4       Key = 0x00_8A
	Key_International5       Key = 0x00_8B
	Key_International6       Key = 0x00_8C
	Key_International7       Key = 0x00_8D
	Key_International8       Key = 0x00_8E
	Key_International9       Key = 0x00_8F
	Key_LANG1                Key = 0x00_90
	Key_LANG2                Key = 0x00_91
	Key_LANG3                Key = 0x00_92
	Key_LANG4                Key = 0x00_93
	Key_LANG5                Key = 0x00_94
	Key_LANG6                Key = 0x00_95
	Key_LANG7                Key = 0x00_96
	Key_LANG8                Key = 0x00_97
	Key_LANG9                Key = 0x00_98
	Key_AlternateErase       Key = 0x00_99
	Key_SysReq               Key = 0x00_9A
	Key_Cancel               Key = 0x00_9B
	Key_Clear                Key = 0x00_9C
	Key_Prior                Key = 0x00_9D
	Key_Return               Key = 0x00_9E
	Key_Separator            Key = 0x00_9F
	Key_Out                  Key = 0x00_A0
	Key_Oper                 Key = 0x00_A1
	Key_ClearAgain           Key = 0x00_A2
	Key_CrSelProps           Key = 0x00_A3
	Key_ExSel                Key = 0x00_A4
)

type Keymap map[byte]Key

var USQwertyKeyboard = Keymap{
	'a': Key_A,
	'b': Key_B,
	'c': Key_C,
	'd': Key_D,
	'e': Key_E,
	'f': Key_F,
	'g': Key_G,
	'h': Key_H,
	'i': Key_I,
	'j': Key_J,
	'k': Key_K,
	'l': Key_L,
	'm': Key_M,
	'n': Key_N,
	'o': Key_O,
	'p': Key_P,
	'q': Key_Q,
	'r': Key_R,
	's': Key_S,
	't': Key_T,
	'u': Key_U,
	'v': Key_V,
	'w': Key_W,
	'x': Key_X,
	'y': Key_Y,
	'z': Key_Z,

	'A': Key_LeftShift | Key_A,
	'B': Key_LeftShift | Key_B,
	'C': Key_LeftShift | Key_C,
	'D': Key_LeftShift | Key_D,
	'E': Key_LeftShift | Key_E,
	'F': Key_LeftShift | Key_F,
	'G': Key_LeftShift | Key_G,
	'H': Key_LeftShift | Key_H,
	'I': Key_LeftShift | Key_I,
	'J': Key_LeftShift | Key_J,
	'K': Key_LeftShift | Key_K,
	'L': Key_LeftShift | Key_L,
	'M': Key_LeftShift | Key_M,
	'N': Key_LeftShift | Key_N,
	'O': Key_LeftShift | Key_O,
	'P': Key_LeftShift | Key_P,
	'Q': Key_LeftShift | Key_Q,
	'R': Key_LeftShift | Key_R,
	'S': Key_LeftShift | Key_S,
	'T': Key_LeftShift | Key_T,
	'U': Key_LeftShift | Key_U,
	'V': Key_LeftShift | Key_V,
	'W': Key_LeftShift | Key_W,
	'X': Key_LeftShift | Key_X,
	'Y': Key_LeftShift | Key_Y,
	'Z': Key_LeftShift | Key_Z,

	'`': Key_Grave,
	'1': Key_1,
	'2': Key_2,
	'3': Key_3,
	'4': Key_4,
	'5': Key_5,
	'6': Key_6,
	'7': Key_7,
	'8': Key_8,
	'9': Key_9,
	'0': Key_0,
	'-': Key_Minus,
	'=': Key_Equal,

	'[':  Key_LeftBracket,
	']':  Key_RightBracket,
	'\\': Key_Backslash,

	';':  Key_Semicolon,
	'\'': Key_Apostrophe,
	'\n': Key_Enter,

	',': Key_Comma,
	'.': Key_Period,
	'/': Key_Slash,

	'~': Key_LeftShift | Key_Grave,
	'!': Key_LeftShift | Key_1,
	'@': Key_LeftShift | Key_2,
	'#': Key_LeftShift | Key_3,
	'$': Key_LeftShift | Key_4,
	'%': Key_LeftShift | Key_5,
	'^': Key_LeftShift | Key_6,
	'&': Key_LeftShift | Key_7,
	'*': Key_LeftShift | Key_8,
	'(': Key_LeftShift | Key_9,
	')': Key_LeftShift | Key_0,
	'_': Key_LeftShift | Key_Minus,
	'+': Key_LeftShift | Key_Equal,

	'{': Key_LeftShift | Key_LeftBracket,
	'}': Key_LeftShift | Key_RightBracket,
	'|': Key_LeftShift | Key_Backslash,

	':': Key_LeftShift | Key_Semicolon,
	'"': Key_LeftShift | Key_Apostrophe,

	'<': Key_LeftShift | Key_Comma,
	'>': Key_LeftShift | Key_Period,
	'?': Key_LeftShift | Key_Slash,

	'\b': Key_Backspace,
	'\t': Key_Tab,
	' ':  Key_Space,
	'\r': Key_Return,
}

func (k Key) Modifier() Modifier {
	if v := k & 0xff_00; v != 0 {
		return Modifier(v >> 8)
	}
	return 0
}

func (k Key) Keycode() byte { return byte(k & 0xff) }

// HID Report
type Report struct {
	Mod      Modifier
	Keycodes [6]byte
}

const ReportLen = 8

func (r *Report) Clear() *Report {
	*r = Report{}
	return r
}

func (r *Report) SetKeys(keys ...Key) *Report {
	var i = 0
	for _, k := range keys {
		if i == len(r.Keycodes) {
			break
		}

		if m := k.Modifier(); m != 0 {
			r.Mod |= m
		}

		if kc := k.Keycode(); kc != 0 {
			r.Keycodes[i] = kc
			i++
		}
	}

	return r
}

func (r *Report) Raw() (b [ReportLen]byte) {
	b[0] = byte(r.Mod)
	copy(b[2:], r.Keycodes[:])
	return
}
