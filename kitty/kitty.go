package kitty

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"io"
	"math"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/disintegration/imaging"
)

// KittyMode defines the rendering mode for the Kitty terminal.
type KittyMode int

const (
	// KittyModeNormal uses the standard Kitty graphics protocol.
	KittyModeNormal KittyMode = iota
	// KittyModeUnicodePlaceholder uses the Unicode placeholder method (for tmux compatibility).
	KittyModeUnicodePlaceholder
)

// Encoder encodes image for Kitty terminal.
type Encoder struct {
	w      io.Writer
	Mode   KittyMode
	Width  int // Optional: force width in pixels
	Height int // Optional: force height in pixels

	// For placeholder mode
	randSource *rand.Rand
	isTmux     bool // Flag to indicate if running under tmux
}

// NewEncoder returns a new Kitty encoder.
func NewEncoder(w io.Writer, mode KittyMode) *Encoder {
	src := rand.NewSource(time.Now().UnixNano())
	isTmux := os.Getenv("TMUX") != ""
	return &Encoder{
		w:          w,
		Mode:       mode,
		randSource: rand.New(src),
		isTmux:     isTmux,
	}
}

const (
	defaultCellWidth  = 8  // Approximate cell width in pixels
	defaultCellHeight = 16 // Approximate cell height in pixels
)

// wrapForTmux wraps a given escape sequence for tmux passthrough.
func (e *Encoder) wrapForTmux(sequence string) string {
	if !e.isTmux {
		return sequence
	}
	// Escape internal ESC characters
	escapedSequence := strings.ReplaceAll(sequence, "\033", "\033\033")
	// Wrap in tmux DCS sequence
	return fmt.Sprintf("\033Ptmux;%s\033\\", escapedSequence)
}

// writeSequence writes the sequence, wrapping for tmux if necessary.
func (e *Encoder) writeSequence(sequence string) (int, error) {
	wrappedSequence := e.wrapForTmux(sequence)
	return e.w.Write([]byte(wrappedSequence))
}

// Encode encodes image to Kitty graphics protocol escape sequences.
func (e *Encoder) Encode(img image.Image) error {
	width, height := img.Bounds().Dx(), img.Bounds().Dy()
	if width == 0 || height == 0 {
		return nil
	}
	max_size := 10000
	if width > max_size || height > max_size {
		if width > height {
			width = max_size
			height = 0 // preserve aspect ratio
		} else {
			width = 0 // preserve aspect ratio
			height = max_size
		}
		img = imaging.Resize(img, width, height, imaging.Lanczos)
	}
	if e.Width != 0 {
		width = e.Width
	}
	if e.Height != 0 {
		height = e.Height
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return err
	}
	b64data := base64.StdEncoding.EncodeToString(buf.Bytes())

	switch e.Mode {
	case KittyModeUnicodePlaceholder:
		return e.encodeUnicodePlaceholder(b64data, width, height)
	default: // KittyModeNormal
		return e.encodeNormal(b64data, width, height)
	}
}

// encodeNormal sends the image using the standard Kitty graphics protocol.
func (e *Encoder) encodeNormal(b64data string, width, height int) error {
	chunk_size := 4096
	var builder strings.Builder
	for i := 0; i < int(math.Ceil(float64(len(b64data))/float64(chunk_size))); i++ {
		if i == 0 {
			builder.WriteString(fmt.Sprintf("\033_Ga=T,f=100,s=%d,v=%d,", width, height))
		} else {
			builder.WriteString("\033_G")
		}
		if (i+1)*chunk_size < len(b64data) {
			builder.WriteString(fmt.Sprintf("m=1;%s\033\\", b64data[i*chunk_size:(i+1)*chunk_size]))
		} else {
			builder.WriteString(fmt.Sprintf("m=0;%s\033\\\n", b64data[i*chunk_size:]))
		}
	}
	e.w.Write([]byte(builder.String()))
	return nil
}

// encodeUnicodePlaceholder sends the image using the Unicode placeholder method.
func (e *Encoder) encodeUnicodePlaceholder(b64data string, width, height int) error {
	// 1. Generate Random 32-bit ID
	imgID := uint32(0)
	for imgID == 0 {
		imgID = e.randSource.Uint32()
	}

	// 2. Calculate Cell Dimensions (approximate)
	cols := (width + defaultCellWidth - 1) / defaultCellWidth
	rows := (height + defaultCellHeight - 1) / defaultCellHeight
	if cols == 0 {
		cols = 1
	}
	if rows == 0 {
		rows = 1
	}

	// Check if image is too large for diacritic encoding (max 256 rows/cols)
	if rows > 256 || cols > 256 {
		return fmt.Errorf("image too large for Unicode placeholder (%dx%d cells, max 256x256)", cols, rows)
	}

	// 3. Transfer Image Data and Set Virtual Placement in the *first* chunk command
	chunk_size := 4096
	var transferSequence strings.Builder // Build sequence before potential wrapping

	for i := 0; i < int(math.Ceil(float64(len(b64data))/float64(chunk_size))); i++ {
		chunk := b64data[i*chunk_size : min((i+1)*chunk_size, len(b64data))]
		more := 0
		if (i+1)*chunk_size < len(b64data) {
			more = 1
		}

		// Build the APC sequence for this chunk
		var chunkBuilder strings.Builder
		if i == 0 {
			// First chunk: Combine a=T, U=1, and placement parameters (c, r)
			// Removing q=2 to see if terminal response is necessary for placeholder registration.
			chunkBuilder.WriteString(fmt.Sprintf(
				"\033_Ga=T,U=1,q=2,i=%d,c=%d,r=%d,f=100,s=%d,v=%d,m=%d;",
				imgID, cols, rows, width, height, more))
		} else {
			// Subsequent chunks only need m=...
			chunkBuilder.WriteString(fmt.Sprintf("\033_Gm=%d;", more))
		}
		chunkBuilder.WriteString(chunk)
		chunkBuilder.WriteString("\033\\")

		// Append the potentially wrapped chunk sequence
		transferSequence.WriteString(e.wrapForTmux(chunkBuilder.String()))
	}
	// Write the complete (potentially wrapped) transfer/placement sequence
	if _, err := e.w.Write([]byte(transferSequence.String())); err != nil {
		return fmt.Errorf("failed to transfer image data/placement: %w", err)
	}

	// 4. Output Placeholder Grid
	var placeholderGrid strings.Builder
	r_id := (imgID >> 16) & 0xFF
	g_id := (imgID >> 8) & 0xFF
	b_id := imgID & 0xFF
	sgrColor := fmt.Sprintf("\033[38;2;%d;%d;%dm", r_id, g_id, b_id)
	placeholderChar := "\U0010EEEE"
	idMsbDiacritic := getDiacritic(int((imgID >> 24) & 0xFF))

	for row := 0; row < rows; row++ {
		placeholderGrid.WriteString(sgrColor)
		rowDiacritic := getDiacritic(row)
		for col := 0; col < cols; col++ {
			colDiacritic := getDiacritic(col)
			placeholderGrid.WriteString(placeholderChar)
			placeholderGrid.WriteString(rowDiacritic)
			placeholderGrid.WriteString(colDiacritic)
			placeholderGrid.WriteString(idMsbDiacritic)
		}
		placeholderGrid.WriteString("\033[39m")
		placeholderGrid.WriteString("\n")
	}

	// Write the placeholder grid directly (NOT wrapped for tmux)
	_, err := e.w.Write([]byte(placeholderGrid.String()))
	return err
}
