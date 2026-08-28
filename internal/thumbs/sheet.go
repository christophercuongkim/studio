package thumbs

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
)

// Contact-sheet layout.
const (
	thumbW, thumbH = 320, 180
	captionH       = 16
	pad            = 6
)

// contactSheet builds a grid image of the given frames (already candidate PNGs),
// cols wide, each captioned with its timestamp.
func contactSheet(frames []Frame, cols int) (*image.RGBA, error) {
	if cols < 1 {
		cols = 1
	}
	rows := (len(frames) + cols - 1) / cols
	cellW, cellH := thumbW+pad, thumbH+captionH+pad
	sheet := image.NewRGBA(image.Rect(0, 0, cols*cellW+pad, rows*cellH+pad))
	draw.Draw(sheet, sheet.Bounds(), &image.Uniform{color.RGBA{20, 22, 26, 255}}, image.Point{}, draw.Src)

	for i, f := range frames {
		src, err := loadPNG(f.Path)
		if err != nil {
			return nil, err
		}
		col, row := i%cols, i/cols
		x0 := pad + col*cellW
		y0 := pad + row*cellH
		thumb := scaleNearest(src, thumbW, thumbH)
		draw.Draw(sheet, image.Rect(x0, y0, x0+thumbW, y0+thumbH), thumb, image.Point{}, draw.Src)
		drawText(sheet, x0+2, y0+thumbH+3, fmt.Sprintf("%d %s", i+1, mmss(f.TimestampSec)), color.RGBA{230, 230, 235, 255})
	}
	return sheet, nil
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func loadPNG(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return png.Decode(f)
}

// scaleNearest resizes src to w×h with nearest-neighbor sampling (stdlib only).
func scaleNearest(src image.Image, w, h int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	b := src.Bounds()
	sw, sh := b.Dx(), b.Dy()
	for y := range h {
		sy := b.Min.Y + y*sh/h
		for x := range w {
			sx := b.Min.X + x*sw/w
			dst.Set(x, y, src.At(sx, sy))
		}
	}
	return dst
}

// mmss formats seconds as M:SS for captions.
func mmss(sec float64) string {
	s := int(sec + 0.5)
	return fmt.Sprintf("%d:%02d", s/60, s%60)
}

// --- tiny 3×5 bitmap font for digits and ':' (no external font dep) ---

var glyphs = map[rune][5]uint8{
	'0': {0b111, 0b101, 0b101, 0b101, 0b111},
	'1': {0b010, 0b110, 0b010, 0b010, 0b111},
	'2': {0b111, 0b001, 0b111, 0b100, 0b111},
	'3': {0b111, 0b001, 0b111, 0b001, 0b111},
	'4': {0b101, 0b101, 0b111, 0b001, 0b001},
	'5': {0b111, 0b100, 0b111, 0b001, 0b111},
	'6': {0b111, 0b100, 0b111, 0b101, 0b111},
	'7': {0b111, 0b001, 0b010, 0b010, 0b010},
	'8': {0b111, 0b101, 0b111, 0b101, 0b111},
	'9': {0b111, 0b101, 0b111, 0b001, 0b111},
	':': {0b000, 0b010, 0b000, 0b010, 0b000},
	' ': {0, 0, 0, 0, 0},
}

// drawText renders s at (x,y) using the 3×5 font scaled 2×.
func drawText(dst *image.RGBA, x, y int, s string, c color.Color) {
	const scale = 2
	cx := x
	for _, r := range s {
		g, ok := glyphs[r]
		if !ok {
			cx += 4 * scale
			continue
		}
		for row := range 5 {
			bits := g[row]
			for col := range 3 {
				if bits&(1<<(2-col)) != 0 {
					for dy := range scale {
						for dx := range scale {
							dst.Set(cx+col*scale+dx, y+row*scale+dy, c)
						}
					}
				}
			}
		}
		cx += 4 * scale
	}
}
