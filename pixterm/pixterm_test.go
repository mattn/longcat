package pixterm

import (
	"errors"
	"image"
	"testing"
)

type errWriter struct{}

func (errWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

func TestEncodeReturnsWriteError(t *testing.T) {
	enc := NewEncoder(errWriter{}, false)
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	if err := enc.Encode(img); err == nil {
		t.Fatal("Encode returned nil for a writer error")
	}
}
