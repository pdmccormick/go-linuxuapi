package main

import (
	"bufio"
	"errors"
	"flag"
	"io"
	"os"
	"time"

	"go.pdmccormick.com/linuxuapi/usb/usbgadget/hidgadget"
)

func main() {
	var (
		devFlag  = flag.String("dev", "/dev/hidg0", "`path` to hidg device")
		textFlag = flag.String("t", "", "text to type")
		holdFlag = flag.Duration("d", 25*time.Millisecond, "key press hold duration")
	)
	flag.Parse()

	dev, err := hidgadget.OpenDevice(*devFlag, *holdFlag)
	if err != nil {
		panic(err)
	}

	defer dev.Close()

	if text := *textFlag; text != "" {
		if err := dev.TypeText(text+"\n", nil); err != nil {
			panic(err)
		}
	} else {
		var r = bufio.NewReader(os.Stdin)
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				if errors.Is(err, io.EOF) {
					return
				}
			}

			if err := dev.TypeText(line, nil); err != nil {
				panic(err)
			}
		}
	}
}
