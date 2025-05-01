package kitty

import (
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/term"
)

// CheckKittyGraphicsProtocol checks for Kitty graphics protocol support using the query method.
func CheckKittyGraphicsProtocol() bool {
	// Use a unique ID for the query, e.g., based on process ID or random
	queryID := uint32(os.Getpid() & 0xFFFFFFFF) // Example ID
	if queryID == 0 {
		queryID = 1 // Ensure non-zero ID
	}
	// Graphics query command (dummy 1x1 RGB pixel)
	// https://sw.kovidgoyal.net/kitty/graphics-protocol/#querying-support-and-available-transmission-mediums
	graphicsQuery := fmt.Sprintf("\033_Ga=q,i=%d,s=1,v=1,t=d,f=24;AAAA\033\\", queryID)

	// Need raw mode to send/receive control sequences without shell interference
	fd := int(os.Stdout.Fd())
	state, err := term.MakeRaw(fd)
	if err != nil {
		return false // Cannot enter raw mode
	}
	defer term.Restore(fd, state)

	// Wrap graphics query for tmux if necessary
	isTmux := os.Getenv("TMUX") != ""
	if isTmux {
		escapedQuery := strings.ReplaceAll(graphicsQuery, "\033", "\033\033")
		graphicsQuery = fmt.Sprintf("\033Ptmux;%s\033\\", escapedQuery)
	}

	// Write only the graphics query
	_, err = os.Stdout.Write([]byte(graphicsQuery))
	if err != nil {
		return false
	}
	os.Stdout.Sync()

	// Read response with timeout
	responseChan := make(chan string)
	go func() {
		var buf [256]byte
		n, readErr := os.Stdout.Read(buf[:])
		if readErr == nil && n > 0 {
			responseChan <- string(buf[:n])
		} else {
			close(responseChan) // Signal no response or error
		}
	}()

	var response string
	select {
	case resp, ok := <-responseChan:
		if ok {
			response = resp
		}
	case <-time.After(10 * time.Millisecond):
	}

	// Check if the response is the graphics protocol ACK
	// Expected format: \033_Gi=<queryID>;OK\033\ (or an error message)
	expectedGraphicsPrefix := fmt.Sprintf("\033_Gi=%d;", queryID)
	if strings.HasPrefix(response, expectedGraphicsPrefix) && strings.HasSuffix(response, "\033\\") {
		return true // Got a graphics response, assume support
	}
	// If we didn't get the graphics response, assume no support
	return false
}
