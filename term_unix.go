//go:build !windows

package main

import (
	"os"
	"time"

	"golang.org/x/term"
)

func queryTerminal(query string, timeout time.Duration) []byte {
	f, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return nil
	}
	defer f.Close()

	s, err := term.MakeRaw(int(f.Fd()))
	if err == nil {
		defer term.Restore(int(f.Fd()), s)
	}
	_, err = f.Write([]byte(query))
	if err != nil {
		return nil
	}
	f.SetReadDeadline(time.Now().Add(timeout))
	defer f.SetReadDeadline(time.Time{})

	time.Sleep(10 * time.Millisecond)

	var b [100]byte
	n, err := f.Read(b[:])
	if err != nil {
		return nil
	}
	return b[:n]
}
