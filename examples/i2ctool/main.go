package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"strconv"
	"strings"

	"go.pdmccormick.com/linuxuapi/i2c"
)

func fromHexString(noun, value string) uint64 {
	var s = strings.TrimSpace(strings.TrimPrefix(strings.ToLower(value), "0x"))

	if s == "" {
		return 0
	}

	n, err := strconv.ParseUint(s, 16, 64)
	if err != nil {
		log.Fatalf("bad %s hex value `%s`: %s", noun, value, err)
	}

	return n
}

func main() {
	var (
		devFlag   = flag.String("d", "", "character device path `/dev/i2c-NN`")
		addrFlag  = flag.Int("a", -1, "slave `addr`ess")
		regFlag   = flag.Int("r", -1, "register `addr`ess")
		writeFlag = flag.Int("w", -1, "write byte to register")
		dumpFlag  = flag.Bool("dump", false, "dump all registers")
	)

	flag.Parse()

	var (
		devName   = *devFlag
		addr      = uint16(*addrFlag)
		reg       = uint8(*regFlag)
		writeByte = uint8(*writeFlag)
	)

	if devName == "" {
		log.Fatalf("missing `-d` flag")
	}

	if *addrFlag < 0 {
		log.Fatalf("missing `-a` flag")
	}

	if *regFlag < 0 && !*dumpFlag {
		log.Fatalf("missing `-r` flag")
	}

	fmt.Printf("Using device %s, address 0x%02x\n", devName, addr)

	dev, err := i2c.OpenDevice(devName)
	if err != nil {
		log.Fatalf("OpenDevice: %s", err)
	}

	defer dev.Close()

	if v := *writeFlag; v >= 0 {
		fmt.Printf("Write 0x%02x to register 0x%02x\n", writeByte, reg)
		if err := dev.WriteReg(addr, reg, writeByte); err != nil {
			log.Fatalf("WriteReg: %s", err)
		}
	} else if *dumpFlag {
		var (
			buf  [256]byte
			dump = buf[:]
		)

		if err := dev.Txn(addr, nil, dump); err != nil {
			log.Fatalf("Txn: %s", err)
		}

		fmt.Print(hex.Dump(dump))
	} else {
		fmt.Printf("Reading register 0x%02x\n", reg)
		v, err := dev.ReadReg(addr, reg)
		if err != nil {
			log.Fatalf("ReadReg: %s", err)
		}

		fmt.Printf("0x%02x\n", v)
	}
}
