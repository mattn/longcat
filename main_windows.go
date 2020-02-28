// +build windows

package main

import (
	"os"

	"golang.org/x/sys/windows"
)

func setup(enabled *bool) func() {
	var mode uint32
	h := windows.Handle(os.Stdout.Fd())
	if err := windows.GetConsoleMode(h, &mode); err == nil {
		if err := windows.SetConsoleMode(h, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING); err == nil {
			if enabled != nil {
				*enabled = true
			}
			return func() {
				windows.SetConsoleMode(h, mode)
			}
		}
	}
	if enabled != nil {
		*enabled = true
	}
	return func() {}
}
