package gadgetconfig

import (
	"strconv"
)

func boolToIntStr(b bool) string {
	if b {
		return "1"
	} else {
		return "0"
	}
}

// Function
type Function interface {
	GadgetFunctionName() string
	GadgetFunctionCreate() Steps
}

// AcmFunction
type AcmFunction struct {
	Name string
}

var _ Function = (*AcmFunction)(nil)

func (fn *AcmFunction) GadgetFunctionName() string { return "acm." + fn.Name }

func (fn *AcmFunction) GadgetFunctionCreate() Steps { return nil }

// EemFunction
type EemFunction struct {
	Name     string
	DevAddr  string
	HostAddr string
}

var _ Function = (*EemFunction)(nil)

func (fn *EemFunction) GadgetFunctionName() string { return "eem." + fn.Name }

func (fn *EemFunction) GadgetFunctionCreate() Steps {
	return Steps{
		Step{Write, "dev_addr", fn.DevAddr},
		Step{Write, "host_addr", fn.HostAddr},
	}
}

func (fn *EemFunction) Ifname(g *Gadget) string {
	data, _ := g.ReadConfigfsFile("functions", fn.GadgetFunctionName(), "ifname")
	return data
}

// NcmFunction
type NcmFunction struct {
	Name     string
	DevAddr  string
	HostAddr string
}

var _ Function = (*NcmFunction)(nil)

func (fn *NcmFunction) GadgetFunctionName() string { return "ncm." + fn.Name }

func (fn *NcmFunction) GadgetFunctionCreate() Steps {
	return Steps{
		Step{Write, "dev_addr", fn.DevAddr},
		Step{Write, "host_addr", fn.HostAddr},
	}
}

func (fn *NcmFunction) Ifname(g *Gadget) string {
	data, _ := g.ReadConfigfsFile("functions", fn.GadgetFunctionName(), "ifname")
	return data
}

// HidFunction
type HidFunction struct {
	Name         string
	Protocol     int
	Subclass     int
	ReportLength int
	Descriptor   []byte
}

var _ Function = (*HidFunction)(nil)

func (fn *HidFunction) GadgetFunctionName() string { return "hid." + fn.Name }

func (fn *HidFunction) GadgetFunctionCreate() Steps {
	return Steps{
		Step{Write, "protocol", strconv.Itoa(fn.Protocol)},
		Step{Write, "subclass", strconv.Itoa(fn.Subclass)},
		Step{Write, "report_length", strconv.Itoa(fn.ReportLength)},
		Step{WriteBinary, "report_desc", string(fn.Descriptor)},
	}
}

// MassStorageFunction
type MassStorageFunction struct {
	Name string
	Luns []MassStorageLun
}

var _ Function = (*MassStorageFunction)(nil)

func (fn *MassStorageFunction) GadgetFunctionName() string { return "mass_storage." + fn.Name }

func (fn *MassStorageFunction) GadgetFunctionCreate() (steps Steps) {
	for _, lun := range fn.Luns {
		var (
			prefix = "lun." + lun.Name
			lsteps = Steps{
				Step{MkdirCreateOnly, "", ""},
			}
		)

		lsteps.Extend(lun.lunCreate())
		lsteps.PrependPath(prefix)

		steps.Extend(lsteps)
	}

	return
}

// MassStorageLun
type MassStorageLun struct {
	Name      string
	File      string
	Removable bool
	Cdrom     bool
}

func (lun *MassStorageLun) lunCreate() Steps {
	return Steps{
		Step{Write, "file", lun.File},
		Step{Write, "removable", boolToIntStr(lun.Removable)},
		Step{Write, "cdrom", boolToIntStr(lun.Cdrom)},
	}
}
