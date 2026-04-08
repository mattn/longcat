//go:build !windows

package main

import (
	"fmt"
	"image/color"
	"os"
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"
)

func detectBackgroundColor() {
	f, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return
	}
	defer f.Close()

	fd := int(f.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return
	}
	defer term.Restore(fd, oldState)
	syscall.SetNonblock(fd, true)
	f.Write([]byte("\x1b]11;?\x1b\\"))

	var bb []byte
	for {
		f.SetDeadline(time.Now().Add(100 * time.Millisecond))
		var b [1]byte
		n, err := f.Read(b[:])
		if err != nil {
			break
		}
		if n == 0 || b[0] == '\\' || b[0] == 0x0a {
			break
		}
		bb = append(bb, b[0])
	}
	if pos := strings.Index(string(bb), "rgb:"); pos != -1 {
		bb = bb[pos+4:]
		pos = strings.Index(string(bb), "\x1b")
		if pos != -1 {
			bb = bb[:pos]
		}
		var r, g, b uint16
		n, err := fmt.Sscanf(string(bb), "%x/%x/%x", &r, &g, &b)
		if err == nil && n == 3 {
			bgColor = color.RGBA64{r, g, b, 0xFFFF}
		}
	}
}
