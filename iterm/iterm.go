package iterm

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"io"

	"github.com/disintegration/imaging"
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
	if width == 0 || height == 0 {
		return nil
	}
	maxDimension := 9999 // kMaxDimension-1 in iTerm2/sources/iTermImage.m
	if width > maxDimension || height > maxDimension {
		if width > height {
			width = maxDimension
			height = 0 // preserve aspect ratio
		} else {
			width = 0 // preserve aspect ratio
			height = maxDimension
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

	fmt.Fprint(e.w, "\033]1337;")
	fmt.Fprintf(e.w, "File=inline=1")
	fmt.Fprintf(e.w, ";width=%dpx", width)
	fmt.Fprintf(e.w, ";height=%dpx", height)
	fmt.Fprint(e.w, ":")
	fmt.Fprintf(e.w, "%s", base64.StdEncoding.EncodeToString(buf.Bytes()))
	fmt.Fprint(e.w, "\a\n")

	return nil
}
