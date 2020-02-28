// +build !windows

package main

func setup(enabled *bool) func() {
	if enabled != nil {
		*enabled = true
	}
	return func() {}
}
