package pixterm

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/eliukblau/pixterm/pkg/ansimage"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/tomnomnom/xtermcolor"
	"golang.org/x/crypto/ssh/terminal"
)

// Encoder encode image to sixel format
type Encoder struct {
	w           io.Writer
	Width       int
	Height      int
	is8BitColor bool
}

// NewEncoder return new instance of Encoder
func NewEncoder(w io.Writer, is8BitColor bool) *Encoder {
	return &Encoder{w: w, is8BitColor: is8BitColor}
}

func to8BitColor(s string) string {
	re := regexp.MustCompile(`\x1b\[([34]8);2;(\d+);(\d+);(\d+)((;\d+)*)m`)
	found := re.FindAllStringSubmatchIndex(s, -1)
	pos := 0
	var builder strings.Builder
	for i := 0; i < len(found); i++ {
		if pos < found[i][0] {
			builder.WriteString(s[pos:found[i][0]])
		}
		r, _ := strconv.Atoi(s[found[i][4]:found[i][5]])
		g, _ := strconv.Atoi(s[found[i][6]:found[i][7]])
		b, _ := strconv.Atoi(s[found[i][8]:found[i][9]])
		c := xtermcolor.FromColor(color.RGBA{uint8(r), uint8(g), uint8(b), 255})
		builder.WriteString(fmt.Sprintf("\x1b[%s;5;%d%sm", s[found[i][2]:found[i][3]], c, s[found[i][10]:found[i][11]]))
		pos = found[i][1]
	}
	builder.WriteString(s[pos:])
	return builder.String()
}

func (e *Encoder) Encode(img image.Image) error {
	width, height := img.Bounds().Dx(), img.Bounds().Dy()
	if width == 0 || height == 0 {
		return nil
	}
	tw, _, err := terminal.GetSize(int(os.Stdout.Fd()))
	if err == nil && tw > 0 && width > tw*ansimage.BlockSizeX {
		width = tw * ansimage.BlockSizeX
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

	sm := ansimage.ScaleMode(2)
	dm := ansimage.DitheringMode(0)
	mc, err := colorful.Hex("#000000")
	if err != nil {
		return err
	}
	pix, err := ansimage.NewScaledFromReader(&buf, height/4, width/4, mc, sm, dm)
	if err != nil {
		return err
	}
	s := pix.Render()
	if e.is8BitColor {
		s = to8BitColor(s)
	}
	e.w.Write([]byte(s))
	return nil
}
