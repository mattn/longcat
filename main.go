package main

//go:generate go get github.com/rakyll/statik
//go:generate statik

import (
	"bytes"
	"flag"

	//"fmt"
	"image"
	"image/draw"
	"image/png"
	"log"
	"net/http"
	"os"

	"github.com/mattn/go-sixel"
	_ "github.com/mattn/longcat/statik"
	"github.com/rakyll/statik/fs"
)

func loadImage(fs http.FileSystem, n string) (image.Image, error) {
	f, err := fs.Open(n)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	return png.Decode(f)
}

func main() {
	var ncat int
	flag.IntVar(&ncat, "n", 1, "numcat")
	flag.Parse()

	fs, err := fs.New()
	if err != nil {
		log.Fatal(err)
	}

	img1, _ := loadImage(fs, "/data01.png")
	img2, _ := loadImage(fs, "/data02.png")
	img3, _ := loadImage(fs, "/data03.png")

	rect := image.Rect(0, 0, img1.Bounds().Dx(), img1.Bounds().Dy()+img2.Bounds().Dy()*ncat+img3.Bounds().Dy())
	canvas := image.NewRGBA(rect)
	rect = image.Rect(0, 0, img1.Bounds().Dx(), img1.Bounds().Dy())
	draw.Draw(canvas, rect, img1, image.Pt(0, 0), draw.Over)
	for i := 0; i < ncat; i++ {
		rect = image.Rect(0, img1.Bounds().Dy()+img2.Bounds().Dy()*i, img1.Bounds().Dx(), img1.Bounds().Dy()+img2.Bounds().Dy()*(i+1))
		draw.Draw(canvas, rect, img2, image.Pt(0, 0), draw.Over)
	}
	rect = image.Rect(0, img1.Bounds().Dy()+img2.Bounds().Dy()*ncat, img1.Bounds().Dx(), img1.Bounds().Dy()+img2.Bounds().Dy()*ncat+img3.Bounds().Dy())
	draw.Draw(canvas, rect, img3, image.Pt(0, 0), draw.Over)

	var buf bytes.Buffer
	err = sixel.NewEncoder(&buf).Encode(canvas)
	if err != nil {
		log.Fatal(err)
	}
	os.Stdout.Write(buf.Bytes())
	os.Stdout.Sync()
}
