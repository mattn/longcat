package iterm

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
	enc := NewEncoder(errWriter{})
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	if err := enc.Encode(img); err == nil {
		t.Fatal("Encode returned nil for a writer error")
	}
}
