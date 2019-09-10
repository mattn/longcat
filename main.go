package main

//go:generate go get github.com/rakyll/statik
//go:generate statik

import (
	"bytes"
	"flag"
	"io/ioutil"

	"image"
	"image/draw"
	"image/png"
	"log"
	"net/http"
	"os"

	"github.com/disintegration/imaging"
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

func saveImage(filename string, img image.Image) error {
	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, buf.Bytes(), 0644)
}

func main() {
	var nlong int
	var ncolumns int
	var rinterval float64
	var reverseH bool
	var reverseV bool
	var filename string

	flag.IntVar(&nlong, "n", 1, "how long cat")
	flag.IntVar(&ncolumns, "l", 1, "number of columns")
	flag.Float64Var(&rinterval, "i", 1.0, "rate of intervals")
	flag.BoolVar(&reverseH, "r", false, "reverse holizontal")
	flag.BoolVar(&reverseV, "rv", false, "reverse vertical")
	flag.StringVar(&filename, "o", "", "output image file")
	flag.Parse()

	fs, err := fs.New()
	if err != nil {
		log.Fatal(err)
	}

	img1, _ := loadImage(fs, "/data01.png")
	img2, _ := loadImage(fs, "/data02.png")
	img3, _ := loadImage(fs, "/data03.png")

	if reverseH {
		img1 = imaging.FlipH(img1)
		img2 = imaging.FlipH(img2)
		img3 = imaging.FlipH(img3)
	}

	rect := image.Rect(0, 0, img1.Bounds().Dx()*ncolumns, img1.Bounds().Dy()+img2.Bounds().Dy()*nlong+img3.Bounds().Dy())
	canvas := image.NewRGBA(rect)

	for col := 0; col < ncolumns; col++ {
		x := int(float64(img1.Bounds().Dx()*col) * rinterval)
		rect = image.Rect(x, 0, x+img1.Bounds().Dx(), img1.Bounds().Dy())
		draw.Draw(canvas, rect, img1, image.Pt(0, 0), draw.Over)
		for i := 0; i < nlong; i++ {
			rect = image.Rect(x, img1.Bounds().Dy()+img2.Bounds().Dy()*i, x+img1.Bounds().Dx(), img1.Bounds().Dy()+img2.Bounds().Dy()*(i+1))
			draw.Draw(canvas, rect, img2, image.Pt(0, 0), draw.Over)
		}
		rect = image.Rect(x, img1.Bounds().Dy()+img2.Bounds().Dy()*nlong, x+img1.Bounds().Dx(), img1.Bounds().Dy()+img2.Bounds().Dy()*nlong+img3.Bounds().Dy())
		draw.Draw(canvas, rect, img3, image.Pt(0, 0), draw.Over)
	}

	var output image.Image = canvas
	if reverseV {
		output = imaging.FlipV(output)
	}

	if filename != "" {
		err = saveImage(filename, output)
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	var buf bytes.Buffer
	err = sixel.NewEncoder(&buf).Encode(output)
	if err != nil {
		log.Fatal(err)
	}
	os.Stdout.Write(buf.Bytes())
	os.Stdout.Sync()
}
