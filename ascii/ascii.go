package ascii

import (
	"bytes"
	"image"
	"image/png"
	"io"

	"github.com/zyxar/image2ascii/ascii"
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

	_w, _h := width/4, height/9 // By default 31x31.
	a, err := ascii.Decode(&buf, ascii.Options{Width: _w, Height: _h})
	if err != nil {
		return err
	}
	if _, err := a.WriteTo(e.w); err != nil {
		return err
	}
	return nil
}
