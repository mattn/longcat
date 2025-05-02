package kitty

import (
	"log"
	"os"
	"regexp"
	"strconv"
	"time"

	"golang.org/x/term"
)

const (
	defaultCellWidth  = 8
	defaultCellHeight = 16
)

// getCellSize attempts to query the terminal for cell size in pixels.
// Uses CSI 16 t: "Report xterm window character cell size in pixels" -> CSI 6 ; height ; width t
// Returns default values on error.
func getCellSize() (width, height int) {
	// Use defaults initially
	width = defaultCellWidth
	height = defaultCellHeight

	query := "\033[16t"
	// Use stdin for raw mode check/restore, stdout for writing query
	stdinFd := int(os.Stdin.Fd())
	stdoutFd := int(os.Stdout.Fd())

	// Check if stdin/stdout are terminals
	if !term.IsTerminal(stdinFd) || !term.IsTerminal(stdoutFd) {
		log.Printf("Warning: Cannot query cell size: stdin/stdout not a terminal.")
		return
	}

	state, err := term.MakeRaw(stdinFd)
	if err != nil {
		log.Printf("Warning: Cannot query cell size: failed to enter raw mode: %v", err)
		return
	}
	defer term.Restore(stdinFd, state)

	// Write query to stdout
	_, err = os.Stdout.Write([]byte(query))
	if err != nil {
		log.Printf("Warning: Cannot query cell size: failed to write query: %v", err)
		return
	}

	// Read response from stdin with timeout
	responseChan := make(chan string)
	readErrChan := make(chan error)
	go func() {
		var buf [64]byte // Buffer for response
		n, readErr := os.Stdin.Read(buf[:])
		if readErr != nil {
			readErrChan <- readErr
		} else if n > 0 {
			responseChan <- string(buf[:n])
		} else {
			close(responseChan) // Should not happen?
		}
	}()

	var response string
	select {
	case resp := <-responseChan:
		response = resp
	case err = <-readErrChan:
		log.Printf("Warning: Cannot query cell size: failed to read response: %v", err)
		return
	case <-time.After(150 * time.Millisecond): // Increased timeout slightly
		log.Printf("Warning: Cannot query cell size: timeout waiting for response.")
		return
	}

	// Parse response: \033[6;<height>;<width>t
	re := regexp.MustCompile(`\033\[6;(\d+);(\d+)t`)
	matches := re.FindStringSubmatch(response)

	if len(matches) == 3 {
		h, e1 := strconv.Atoi(matches[1])
		w, e2 := strconv.Atoi(matches[2])
		if e1 == nil && e2 == nil && h > 0 && w > 0 {
			width = w
			height = h
			return
		}
	}

	log.Printf("Warning: Cannot query cell size: failed to parse response: %q", response)
	return
}
