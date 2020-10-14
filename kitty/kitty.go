package kitty

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"io"
	"math"
	"strings"

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
