package iterm

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"io"
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

	fmt.Print("\033]1337;")
	fmt.Printf("File=inline=1")
	fmt.Printf(";width=%dpx", width)
	fmt.Printf(";height=%dpx", height)
	fmt.Print(":")
	fmt.Printf("%s", base64.StdEncoding.EncodeToString(buf.Bytes()))
	fmt.Print("\a\n")

	return nil
}
