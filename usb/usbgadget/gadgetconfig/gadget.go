package gadgetconfig

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

const (
	GadgetConfigBasePath = "/sys/kernel/config/usb_gadget"
	UdcPathGlob          = "/sys/class/udc/*"
	StrEnglish           = "0x409"
)

func FindUdc() []string {
	var (
		names []string
		err   error
	)

	if names, err = filepath.Glob(UdcPathGlob); err != nil {
		return nil
	}

	for i, path := range names {
		names[i] = filepath.Base(path)
	}

	return names
}

// Gadget
type Gadget struct {
	Name         string
	GadgetPath   string
	IdVendor     int
	IdProduct    int
	SerialNumber string
	Manufacturer string
	Product      string
	UDC          string
	Configs      []Config
}

func (g *Gadget) gadgetPath() string {
	if v := g.GadgetPath; v != "" {
		return v
	}
	return filepath.Join(GadgetConfigBasePath, g.Name)
}

func (g *Gadget) Exists() bool {
	_, err := os.Stat(g.gadgetPath())
	return !os.IsNotExist(err)
}

func (g *Gadget) Create() error           { return g.CreateSteps().Run() }
func (g *Gadget) Remove() error           { return g.RemoveSteps().Run() }
func (g *Gadget) ShellCreate() ShellSteps { return g.CreateSteps().ShellArgs() }
func (g *Gadget) ShellRemove() ShellSteps { return g.RemoveSteps().ShellArgs() }

func (g *Gadget) RemoveSteps() Steps { return g.CreateSteps().Undo().Reverse() }

func (g *Gadget) CreateSteps() (steps Steps) {
	steps = Steps{
		Step{Mkdir, "", ""},
		Step{Write, "idVendor", fmt.Sprintf("0x%04x", g.IdVendor)},
		Step{Write, "idProduct", fmt.Sprintf("0x%04x", g.IdProduct)},

		Step{Mkdir, "strings/" + StrEnglish, ""},
		Step{Write, "strings/" + StrEnglish + "/serialnumber", g.SerialNumber},
		Step{Write, "strings/" + StrEnglish + "/manufacturer", g.Manufacturer},
		Step{Write, "strings/" + StrEnglish + "/product", g.Product},
	}

	for i := range g.Configs {
		var (
			c           = &g.Configs[i]
			configPath  = "configs/" + c.Name
			configSteps = Steps{
				Step{Comment, fmt.Sprintf("config `%s`", c.Name), ""},
				Step{Mkdir, "", ""},
				Step{Mkdir, "strings/" + StrEnglish, ""},
				Step{Write, "strings/" + StrEnglish + "/configuration", c.Configuration},
			}
		)

		configSteps.PrependPath(configPath)
		steps.Extend(configSteps)

		for _, fn := range c.Functions {
			var (
				name    = fn.GadgetFunctionName()
				fnPath  = "functions/" + name
				fnSteps = Steps{
					Step{Comment, fmt.Sprintf("config `%s`, function `%s`", c.Name, name), ""},
					Step{Mkdir, "", ""},
				}
			)

			fnSteps.Extend(fn.GadgetFunctionCreate())
			fnSteps.PrependPath(fnPath)
			steps.Extend(fnSteps)

			// Attach function to configuration
			steps.Append(Step{Symlink, fnPath, configPath + "/" + name})
		}
	}

	if v := g.UDC; v != "" {
		steps.Append(Step{Write, "UDC", v})
	}

	return steps.PrependPath(g.gadgetPath())
}

func (g *Gadget) ReadConfigfsFile(elem ...string) (string, error) {
	var path = filepath.Join(g.gadgetPath(), filepath.Join(elem...))
	if buf, err := ioutil.ReadFile(path); err != nil {
		return "", nil
	} else {
		return strings.TrimRight(string(buf), "\n"), nil
	}
}

// Config
type Config struct {
	Name          string
	Configuration string
	Functions     []Function
}
