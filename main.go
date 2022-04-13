package main

//go:generate go get github.com/rakyll/statik
//go:generate statik

import (
	"bytes"
	"embed"
	"flag"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
	"github.com/mattn/go-sixel"
	"github.com/mattn/longcat/ascii"
	"github.com/mattn/longcat/extraterm"
	"github.com/mattn/longcat/iterm"
	"github.com/mattn/longcat/kitty"
	"github.com/mattn/longcat/pixterm"
	"golang.org/x/term"
)

const name = "longcat"

const version = "0.0.1"

var revision = "HEAD"

//go:embed public/themes
var themes embed.FS

// Theme of longcat
type Theme struct {
	Head image.Image
	Body image.Image
	Tail image.Image
}

func loadImage(name string) (image.Image, error) {
	f, err := themes.Open(name)
	if err != nil {
		return nil, fmt.Errorf("theme file does not open %s: %w", name, err)
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

	imgPath := func(s string) string {
		return filepath.ToSlash(filepath.Join("public/themes", themeName, s))
	}

	t.Head, err = loadImage(imgPath("data01.png"))
	if err != nil {
		return err
	}
	t.Body, err = loadImage(imgPath("data02.png"))
	if err != nil {
		return err
	}
	t.Tail, err = loadImage(imgPath("data03.png"))
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
	width, height := img.Bounds().Dx(), img.Bounds().Dy()
	if width == 0 || height == 0 {
		return nil
	}
	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, buf.Bytes(), 0644)
}

func getThemeNames() ([]string, error) {
	infos, err := fs.ReadDir(themes, "public/themes")
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

func getDA2() string {
	s, err := term.MakeRaw(1)
	if err != nil {
		return ""
	}
	defer term.Restore(1, s)
	_, err = os.Stdout.Write([]byte("\x1b[>c")) // DA2 host request
	if err != nil {
		return ""
	}
	defer os.Stdout.SetReadDeadline(time.Time{})

	time.Sleep(10 * time.Millisecond)

	var b [100]byte
	n, err := os.Stdout.Read(b[:])
	if err != nil {
		return ""
	}
	return string(b[:n])
}

func checkIterm() bool {
	if strings.HasPrefix(os.Getenv("TERM_PROGRAM"), "iTerm") {
		return true
	}
	return getDA2() == "\x1b[>0;95;0c" // iTerm2 version 3
}

func checkKitty() bool {
	if os.Getenv("KITTY_WINDOW_ID") != "" {
		return true
	}
	return strings.HasPrefix(getDA2(), "\x1b[>1;4000;") // \x1b[>1;{major+4000};{minor}c
}

func checkExtraterm() bool {
	return os.Getenv("LC_EXTRATERM_COOKIE") != ""
}

func check8BitColor() bool {
	if os.Getenv("TERM_PROGRAM") == "Apple_Terminal" { // Terminal.app
		return true
	}
	da2 := getDA2()
	var supportedTerminals = []string{
		"\x1b[>1;95;0c",  // Terminal.app
		"\x1b[>0;276;0c", // tty.js (xterm mode)
	}
	for _, supportedTerminal := range supportedTerminals {
		if da2 == supportedTerminal {
			return true
		}
	}
	return false
}

func checkSixel() bool {
	if isatty.IsCygwinTerminal(os.Stdout.Fd()) {
		return true
	}
	s, err := term.MakeRaw(1)
	if err != nil {
		return false
	}
	defer term.Restore(1, s)
	_, err = os.Stdout.Write([]byte("\x1b[c"))
	if err != nil {
		return false
	}
	defer os.Stdout.SetReadDeadline(time.Time{})

	time.Sleep(10 * time.Millisecond)

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

	sb := b[6:n]
	n = bytes.IndexByte(sb, 'c')
	if n != -1 {
		sb = sb[:n]
	}
	for _, t := range bytes.Split(sb, []byte(";")) {
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
	var is8BitColor bool
	var listsThemes bool
	var darkMode bool
	var asciiMode bool
	var showVersion bool

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
	flag.BoolVar(&is8BitColor, "8", false, "8bit color")
	flag.BoolVar(&listsThemes, "themes", false, "list themes")
	flag.BoolVar(&darkMode, "dark", false, "dark-mode(a.k.a. tacgnol theme)")
	flag.BoolVar(&asciiMode, "ascii", false, "ascii mode")
	flag.BoolVar(&showVersion, "v", false, "print the version")

	flag.Parse()

	if showVersion {
		fmt.Printf("%s %s (rev: %s/%s)\n", name, version, revision, runtime.Version())
		return
	}

	if listsThemes {
		if err := printThemeNames(); err != nil {
			log.Fatal(err)
		}
		return
	}

	var vtenabled bool
	defer colorable.EnableColorsStdout(&vtenabled)()

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
		if runtime.GOOS == "windows" && !isatty.IsCygwinTerminal(os.Stdout.Fd()) {
			if vtenabled {
				isPixterm = true
			} else {
				asciiMode = true
			}
		} else if checkIterm() {
			enc = iterm.NewEncoder(&buf)
		} else if checkKitty() {
			enc = kitty.NewEncoder(&buf)
		} else if checkSixel() {
			enc = sixel.NewEncoder(&buf)
		} else if checkExtraterm() {
			enc = extraterm.NewEncoder(&buf)
		} else {
			isPixterm = true
		}
	}

	if isPixterm {
		is8BitColor = is8BitColor || check8BitColor()
		enc = pixterm.NewEncoder(&buf, is8BitColor)
	}

	if asciiMode {
		enc = ascii.NewEncoder(&buf)
	}

	if err := enc.Encode(output); err != nil {
		log.Fatal(err)
	}

	if runtime.GOOS == "windows" {
		colorable.NewColorableStdout().Write(buf.Bytes())
	} else {
		os.Stdout.Write(buf.Bytes())
	}
	os.Stdout.Sync()
}
