package kitty

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// QueryTerminal is a hook for sending a query to the controlling terminal and
// reading its response. It is set by the main package to the shared, robust
// /dev/tty based implementation so this package does not need its own (and
// platform specific) terminal I/O. When it is nil, callers fall back to safe
// defaults.
var QueryTerminal func(query string, timeout time.Duration) []byte

// tmuxPassthrough wraps an escape sequence so that tmux forwards it to the
// outer terminal. It requires tmux's "allow-passthrough" option to be enabled;
// otherwise tmux silently drops the sequence.
func tmuxPassthrough(sequence string) string {
	escaped := strings.ReplaceAll(sequence, "\033", "\033\033")
	return fmt.Sprintf("\033Ptmux;%s\033\\", escaped)
}

// CheckKittyGraphicsProtocol reports whether the terminal supports the Kitty
// graphics protocol by sending a query action (a=q) and looking for the
// matching response. This is primarily useful inside tmux, where DA2 reports
// tmux's own identity instead of the outer terminal's.
func CheckKittyGraphicsProtocol() bool {
	if QueryTerminal == nil {
		return false
	}

	queryID := uint32(os.Getpid())
	if queryID == 0 {
		queryID = 1
	}

	// Query support using a dummy 1x1 RGB pixel. See
	// https://sw.kovidgoyal.net/kitty/graphics-protocol/#querying-support-and-available-transmission-mediums
	query := fmt.Sprintf("\033_Ga=q,i=%d,s=1,v=1,t=d,f=24;AAAA\033\\", queryID)
	if os.Getenv("TMUX") != "" {
		query = tmuxPassthrough(query)
	}

	// tmux passthrough adds a round trip, so allow a generous timeout.
	resp := string(QueryTerminal(query, 500*time.Millisecond))
	if resp == "" {
		return false
	}

	// A supporting terminal replies with \033_Gi=<id>;OK\033\\ (or an error
	// payload, which still means the protocol is understood). The response may
	// be preceded by other report bytes, so match the marker anywhere.
	return strings.Contains(resp, fmt.Sprintf("\033_Gi=%d;", queryID))
}
