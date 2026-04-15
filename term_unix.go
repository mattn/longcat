//go:build !windows

package main

import (
	"os"
	"time"

	"golang.org/x/term"
)

func queryTerminal(query string, timeout time.Duration) []byte {
	s, err := term.MakeRaw(1)
	if err == nil {
		defer term.Restore(1, s)
	}
	_, err = os.Stdout.Write([]byte(query))
	if err != nil {
		return nil
	}
	os.Stdout.SetReadDeadline(time.Now().Add(timeout))
	defer os.Stdout.SetReadDeadline(time.Time{})

	time.Sleep(10 * time.Millisecond)

	var b [100]byte
	n, err := os.Stdout.Read(b[:])
	if err != nil {
		return nil
	}
	return b[:n]
}
