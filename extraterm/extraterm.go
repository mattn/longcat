package extraterm

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"io"
	"math"
	"os"
	"strings"
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

	data := buf.Bytes()
	chunk_size := 3 * 1024
	previous_hash := []byte{}
	var builder strings.Builder
	json := "{\"filename\": \"longcat\"}"
	builder.WriteString(fmt.Sprintf("\033&%s;5;%d\a%s", os.Getenv("LC_EXTRATERM_COOKIE"), len(json), json))
	for i := 0; i < int(math.Ceil(float64(len(data))/float64(chunk_size))); i++ {
		var chunk []byte
		if (i+1)*chunk_size < len(data) {
			chunk = data[i*chunk_size : (i+1)*chunk_size]
		} else {
			chunk = data[i*chunk_size:]
		}
		sha256hash := sha256.New()
		sha256hash.Write(previous_hash)
		sha256hash.Write(chunk)
		hash := sha256hash.Sum(nil)
		builder.WriteString(fmt.Sprintf("D:%s:%x\n", base64.StdEncoding.EncodeToString(chunk), hash))
		previous_hash = hash
	}
	builder.WriteString(fmt.Sprintf("E::%x\n\000", sha256.Sum256(previous_hash)))
	e.w.Write([]byte(builder.String()))
	return nil
}
