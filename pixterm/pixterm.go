package pixterm

import (
	"bytes"
	"image"
	"image/png"
	"io"

	"github.com/eliukblau/pixterm/ansimage"
	"github.com/lucasb-eyer/go-colorful"
)

// Encoder encode image to sixel format
type Encoder struct {
	w      io.Writer
	Width  int
	Height int
}

// NewEncoder return new instance of Encoder
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: w}
}

func (e *Encoder) Encode(img image.Image) error {
	width, height := img.Bounds().Dx(), img.Bounds().Dy()
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

	sm := ansimage.ScaleMode(2)
	dm := ansimage.DitheringMode(0)
	mc, err := colorful.Hex("#000000")
	if err != nil {
		return err
	}
	pix, err := ansimage.NewScaledFromReader(&buf, height/4, width/4, mc, sm, dm)
	if err != nil {
		return err
	}
	e.w.Write([]byte(pix.Render()))
	return nil
}
