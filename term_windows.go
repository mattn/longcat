//go:build windows

package main

import (
	"bytes"
	"os"
	"time"

	"golang.org/x/sys/windows"
)

const (
	enableVirtualTerminalInput      = 0x0200
	enableVirtualTerminalProcessing = 0x0004
)

func queryTerminal(query string, timeout time.Duration) []byte {
	hIn := windows.Handle(os.Stdin.Fd())
	hOut := windows.Handle(os.Stdout.Fd())

	var origIn, origOut uint32
	if err := windows.GetConsoleMode(hIn, &origIn); err != nil {
		return nil
	}
	if err := windows.GetConsoleMode(hOut, &origOut); err != nil {
		return nil
	}
	defer windows.SetConsoleMode(hIn, origIn)
	defer windows.SetConsoleMode(hOut, origOut)

	if err := windows.SetConsoleMode(hIn, enableVirtualTerminalInput); err != nil {
		return nil
	}
	if err := windows.SetConsoleMode(hOut, origOut|enableVirtualTerminalProcessing); err != nil {
		return nil
	}

	windows.FlushConsoleInputBuffer(hIn)

	q := []byte(query)
	var written uint32
	if err := windows.WriteFile(hOut, q, &written, nil); err != nil {
		return nil
	}

	deadline := time.Now().Add(timeout)
	var result []byte
	var buf [128]byte
	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			break
		}
		ev, err := windows.WaitForSingleObject(hIn, uint32(remaining/time.Millisecond)+1)
		if err != nil || ev != windows.WAIT_OBJECT_0 {
			break
		}
		var n uint32
		if err := windows.ReadFile(hIn, buf[:], &n, nil); err != nil || n == 0 {
			break
		}
		result = append(result, buf[:n]...)
		if bytes.IndexByte(result, 'c') != -1 {
			break
		}
	}
	return result
}
