package main

import (
	_ "embed"
	"flag"
	"fmt"
	"log"
	"os"

	"go.pdmccormick.com/linuxuapi/usb/usbgadget/gadgetconfig"
)

// See https://github.com/qlyoung/keyboard-gadget/blob/master/gadget-setup.sh

//go:embed kbd-descriptor.bin
var kbdDescriptor []byte

const DefaultMassStorageBackingFile = "/tmp/disk0.img"

var (
	netFunc = gadgetconfig.NcmFunction{
		Name:     "gadgetnet",
		DevAddr:  "16:76:99:89:44:cd",
		HostAddr: "26:f8:5e:d8:ce:42",
	}
	kbdFunc = gadgetconfig.HidFunction{
		Name:         "kbd",
		Protocol:     1,
		Subclass:     1,
		ReportLength: 8,
		Descriptor:   kbdDescriptor,
	}
	massFunc = gadgetconfig.MassStorageFunction{
		Name: "disk",
		Luns: []gadgetconfig.MassStorageLun{
			gadgetconfig.MassStorageLun{
				Name:      "0",
				File:      DefaultMassStorageBackingFile,
				Removable: false,
				Cdrom:     false,
			},
		},
	}
	gadget = gadgetconfig.Gadget{
		Name:         "g1",
		IdVendor:     0x1d6b, // The Linux Foundation
		IdProduct:    0x0104, // Multifunction Composite Gadget
		SerialNumber: "0123456789",
		Manufacturer: "Example Manufacturer",
		Product:      "Example Product",
		Configs: []gadgetconfig.Config{
			gadgetconfig.Config{
				Name:          "c.1",
				Configuration: "Example Config",
				Functions: []gadgetconfig.Function{
					&netFunc,
					&kbdFunc,
					&massFunc,
				},
			},
		},
	}
)

var (
	logf   = log.Printf
	fatalf = log.Fatalf
)

func checkBackingFile() {
	var name = massFunc.Luns[0].File

	if _, err := os.Stat(name); !os.IsNotExist(err) {
		// exists, we're done
		return
	}

	const SizeKB = 64

	logf("backing file `%s` does not exist, attempting to create %d KiB file", name, SizeKB)

	f, err := os.Create(name)
	if err != nil {
		fatalf("error creating `%s`: %s", name, err)
	}

	defer f.Close()
	if err := f.Truncate(SizeKB << 10); err != nil {
		fatalf("error sizing `%s`: %s", name, err)
	}
}

func main() {
	var (
		mkFlag          = flag.Bool("mk", false, "creates gadget")
		rmFlag          = flag.Bool("rm", false, "removes gadget")
		shellFlag       = flag.Bool("sh", false, "show operational steps as a sequence of shell commands")
		udcFlag         = flag.String("udc", "", "use a specific UDC value")
		showUdcFlag     = flag.Bool("showudc", false, "show all possible UDC values")
		storageFileFlag = flag.String("storagefile", DefaultMassStorageBackingFile, "mass storage device backing file")
	)

	flag.Parse()

	massFunc.Luns[0].File = *storageFileFlag

	switch {
	case *showUdcFlag:
		udcs := gadgetconfig.FindUdc()

		if len(udcs) == 0 {
			fatalf("no UDC found")
		}

		for _, v := range udcs {
			fmt.Println(v)
		}

	case *mkFlag:
		var udc = *udcFlag

		if udc == "" {
			udcs := gadgetconfig.FindUdc()

			if len(udcs) == 0 {
				fatalf("no UDC found")
			}

			udc = udcs[0]
		}

		gadget.UDC = udc
		gadget.SerialNumber = "1234"

		if *shellFlag {
			gadget.ShellCreate().Dump(os.Stdout)
		} else {
			if gadget.Exists() {
				fatalf("cannot create, already exists!")
			}

			checkBackingFile()

			if err := gadget.Create(); err != nil {
				fatalf("error creating: %s", err)
			}

			logf("created")
		}

	case *rmFlag:
		if *shellFlag {
			gadget.ShellRemove().Dump(os.Stdout)
		} else {
			if !gadget.Exists() {
				fatalf("cannot remove, does not exist!")
			}

			if err := gadget.Remove(); err != nil {
				fatalf("error removing: %s", err)
			}

			logf("removed")
		}

	default:
		fatalf("must specify either `-mk` or `-rm`")
	}
}
