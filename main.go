package main

//go:generate go get github.com/rakyll/statik
//go:generate statik

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"time"

	"golang.org/x/crypto/ssh/terminal"

	"image"
	"image/draw"
	"image/png"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/mattn/go-isatty"
	"github.com/mattn/go-sixel"
	"github.com/mattn/longcat/iterm"
	"github.com/mattn/longcat/pixterm"
	_ "github.com/mattn/longcat/statik"
	"github.com/rakyll/statik/fs"
)

// Theme of longcat
type Theme struct {
	Head image.Image
	Body image.Image
	Tail image.Image
}

func loadImage(fs http.FileSystem, n string) (image.Image, error) {
	f, err := fs.Open(n)
	if err != nil {
		return nil, fmt.Errorf("theme file does not open: %s", n)
	}
	defer f.Close()
	return png.Decode(f)
}

func loadImageGlob(dir string, glob string) (image.Image, error) {
	pattern := filepath.Join(dir, glob)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("file does not exist: %s", pattern)
	}

	data, err := ioutil.ReadFile(matches[0])
	if err != nil {
		return nil, err
	}

	return png.Decode(bytes.NewReader(data))
}

func (t *Theme) loadTheme(themeName string) error {
	contains := func(ss []string, target string) bool {
		for _, s := range ss {
			if s == target {
				return true
			}
		}
		return false
	}

	// The theme exists?
	themeNames, err := getThemeNames()
	if err != nil {
		return err
	}
	if !contains(themeNames, themeName) {
		return fmt.Errorf("theme does not exist: %s", themeName)
	}

	fs, err := fs.New()
	if err != nil {
		return err
	}

	imgPath := func(s string) string {
		return filepath.ToSlash(filepath.Join("/themes", themeName, s))
	}

	t.Head, err = loadImage(fs, imgPath("data01.png"))
	if err != nil {
		return err
	}
	t.Body, err = loadImage(fs, imgPath("data02.png"))
	if err != nil {
		return err
	}
	t.Tail, err = loadImage(fs, imgPath("data03.png"))
	if err != nil {
		return err
	}

	return nil
}

func (t *Theme) loadThemeFromDir(dir string) error {
	var err error

	t.Head, err = loadImageGlob(dir, "*1.png")
	if err != nil {
		return err
	}
	t.Body, err = loadImageGlob(dir, "*2.png")
	if err != nil {
		return err
	}
	t.Tail, err = loadImageGlob(dir, "*3.png")
	if err != nil {
		return err
	}

	return nil
}

func saveImage(filename string, img image.Image) error {
	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, buf.Bytes(), 0644)
}

func getThemeNames() ([]string, error) {
	fs, err := fs.New()
	if err != nil {
		return nil, err
	}

	f, err := fs.Open("/themes")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// List "/themes/*" directory in statik
	infos, err := f.Readdir(-1)
	if err != nil {
		return nil, err
	}

	names := make([]string, len(infos))
	for i, v := range infos {
		names[i] = v.Name()
	}

	return names, nil
}

func printThemeNames() error {
	names, err := getThemeNames()
	if err != nil {
		return err
	}

	for _, v := range names {
		fmt.Println(v)
	}
	return nil
}

func checkIterm() bool {
	return strings.HasPrefix(os.Getenv("TERM_PROGRAM"), "iTerm")
}

func checkSixel() bool {
	if isatty.IsCygwinTerminal(os.Stdout.Fd()) {
		return true
	}
	s, err := terminal.MakeRaw(1)
	if err != nil {
		return false
	}
	defer terminal.Restore(1, s)
	_, err = os.Stdout.Write([]byte("\x1b[c"))
	if err != nil {
		return false
	}
	defer os.Stdout.SetReadDeadline(time.Time{})

	var b [100]byte
	n, err := os.Stdout.Read(b[:])
	if err != nil {
		return false
	}
	var supportedTerminals = []string{
		"\x1b[?62;", // VT240
		"\x1b[?63;", // wsltty
		"\x1b[?64;", // mintty
		"\x1b[?65;", // RLogin
	}
	supported := false
	for _, supportedTerminal := range supportedTerminals {
		if bytes.HasPrefix(b[:n], []byte(supportedTerminal)) {
			supported = true
			break
		}
	}
	if !supported {
		return false
	}
	for _, t := range bytes.Split(b[6:n], []byte(";")) {
		if len(t) == 1 && t[0] == '4' {
			return true
		}
	}
	return false
}

func main() {
	var nlong int
	var ncolumns int
	var rinterval float64
	var flipH bool
	var flipV bool
	var isHorizontal bool
	var filename string
	var imageDir string
	var themeName string
	var isPixterm bool
	var listsThemes bool
	var darkMode bool

	flag.IntVar(&nlong, "n", 1, "how long cat")
	flag.IntVar(&ncolumns, "l", 1, "number of columns")
	flag.Float64Var(&rinterval, "i", 1.0, "rate of intervals")
	flag.BoolVar(&flipH, "r", false, "flip holizontal")
	flag.BoolVar(&flipV, "R", false, "flip vertical")
	flag.BoolVar(&isHorizontal, "H", false, "holizontal-mode")
	flag.StringVar(&filename, "o", "", "output image file")
	flag.StringVar(&imageDir, "d", "", "directory of images(dir/*{1,2,3}.png)")
	flag.StringVar(&themeName, "t", "longcat", "name of theme")
	flag.BoolVar(&isPixterm, "pixterm", false, "pixterm mode")
	flag.BoolVar(&listsThemes, "themes", false, "list themes")
	flag.BoolVar(&darkMode, "dark", false, "dark-mode(a.k.a. tacgnol theme)")

	flag.Parse()

	if nlong < 1 {
		log.SetFlags(0)
		log.Fatalf("invalid value \"%v\" for flag -n", nlong)
	}
	
	if ncolumns < 1 {
		log.SetFlags(0)
		log.Fatalf("invalid value \"%v\" for flag -l", ncolumns)
	}
	
	if listsThemes {
		if err := printThemeNames(); err != nil {
			log.Fatal(err)
		}
		return
	}

	if darkMode {
		themeName = "tacgnol"
		imageDir = "" // Forcibly apply the above theme
	}

	theme := Theme{}
	if imageDir == "" {
		err := theme.loadTheme(themeName)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		err := theme.loadThemeFromDir(imageDir)
		if err != nil {
			log.Fatal(err)
		}
	}
	img1 := theme.Head
	img2 := theme.Body
	img3 := theme.Tail

	if flipH {
		img1 = imaging.FlipH(img1)
		img2 = imaging.FlipH(img2)
		img3 = imaging.FlipH(img3)
	}

	width := int(float64(img1.Bounds().Dx()*(ncolumns-1))*rinterval) + img1.Bounds().Dx()
	height := img1.Bounds().Dy() + img2.Bounds().Dy()*nlong + img3.Bounds().Dy()
	rect := image.Rect(0, 0, width, height)
	canvas := image.NewRGBA(rect)

	for col := 0; col < ncolumns; col++ {
		x := int(float64(img1.Bounds().Dx()*col) * rinterval)
		rect = image.Rect(x, 0, x+img1.Bounds().Dx(), img1.Bounds().Dy())
		draw.Draw(canvas, rect, img1, image.ZP, draw.Over)
		for i := 0; i < nlong; i++ {
			rect = image.Rect(x, img1.Bounds().Dy()+img2.Bounds().Dy()*i, x+img1.Bounds().Dx(), img1.Bounds().Dy()+img2.Bounds().Dy()*(i+1))
			draw.Draw(canvas, rect, img2, image.ZP, draw.Over)
		}
		rect = image.Rect(x, img1.Bounds().Dy()+img2.Bounds().Dy()*nlong, x+img1.Bounds().Dx(), img1.Bounds().Dy()+img2.Bounds().Dy()*nlong+img3.Bounds().Dy())
		draw.Draw(canvas, rect, img3, image.ZP, draw.Over)
	}

	var output image.Image = canvas
	if flipV {
		output = imaging.FlipV(output)
	}

	output = imaging.Resize(output, width/3, height/3, imaging.Lanczos)

	if isHorizontal {
		output = imaging.Rotate90(output)
	}

	if filename != "" {
		err := saveImage(filename, output)
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	var buf bytes.Buffer
	var enc interface {
		Encode(image.Image) error
	}

	if !isPixterm {
		if checkIterm() {
			enc = iterm.NewEncoder(&buf)
		} else if checkSixel() {
			enc = sixel.NewEncoder(&buf)
		} else {
			isPixterm = true
		}
	}

	if isPixterm {
		enc = pixterm.NewEncoder(&buf)
	}

	if err := enc.Encode(output); err != nil {
		log.Fatal(err)
	}
	os.Stdout.Write(buf.Bytes())
	os.Stdout.Sync()
}
