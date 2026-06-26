package main

import (
	"errors"
	"io"
	"os"
	"testing"
)

var themeDir = "./public/themes"
var imageNames = [...]string{"data01.png", "data02.png", "data03.png"}

func TestThemeImages(t *testing.T) {
	var err error

	_, err = os.Stat(themeDir)
	if err != nil {
		t.Fatal("themes file is not found : ", err)
	}

	themedirs, err := os.ReadDir(themeDir)
	if err != nil {
		t.Fatal("themes file is not found : ", err)
	}

	for _, themedir := range themedirs {
		if themedir.IsDir() {
			t.Log("found theme : ", themedir.Name())
			files, err := os.ReadDir(themeDir + "/" + themedir.Name())
			if err != nil {
				t.Log("theme image is not found. skipping,,, : ", err)
				continue
			}
			chkflg := [len(imageNames)]bool{}
			for _, file := range files {
				if !file.IsDir() {
					for i, v := range imageNames {
						if file.Name() == v {
							chkflg[i] = true
						}
					}
				}
			}
			for i, v := range chkflg {
				if !v {
					t.Error(imageNames[i] + " is not found in " + themedir.Name() + "theme")
				}
			}

		}
	}
}

type shortWriter struct{}

func (shortWriter) Write(p []byte) (int, error) {
	return len(p) - 1, nil
}

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

func TestWriteOutput(t *testing.T) {
	if err := writeOutput(io.Discard, []byte("longcat")); err != nil {
		t.Fatalf("writeOutput returned unexpected error: %v", err)
	}
	if err := writeOutput(failWriter{}, []byte("longcat")); err == nil {
		t.Fatal("writeOutput returned nil for a writer error")
	}
	if err := writeOutput(shortWriter{}, []byte("longcat")); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("writeOutput returned %v, want %v", err, io.ErrShortWrite)
	}
}
