package kitty

import (
	"regexp"
	"strconv"
	"time"
)

const (
	defaultCellWidth  = 8
	defaultCellHeight = 16
)

var cellSizeRe = regexp.MustCompile(`\033\[6;(\d+);(\d+)t`)

// getCellSize queries the terminal for its cell size in pixels using CSI 16 t
// ("report cell size"), which replies with CSI 6 ; height ; width t. It returns
// the default cell size when the query is unavailable or fails.
func getCellSize() (width, height int) {
	width, height = defaultCellWidth, defaultCellHeight
	if QueryTerminal == nil {
		return
	}

	resp := string(QueryTerminal("\033[16t", 200*time.Millisecond))
	matches := cellSizeRe.FindStringSubmatch(resp)
	if len(matches) == 3 {
		h, e1 := strconv.Atoi(matches[1])
		w, e2 := strconv.Atoi(matches[2])
		if e1 == nil && e2 == nil && h > 0 && w > 0 {
			width, height = w, h
		}
	}
	return
}
