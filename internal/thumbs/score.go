// Package thumbs extracts and ranks thumbnail candidate frames from a render or
// from top-rated clips (plan §14). Scoring is pure Go on the luma plane — no
// OpenCV, no third-party image deps.
package thumbs

import "image"

// Frame is one extracted candidate.
type Frame struct {
	Path         string
	TimestampSec float64
	Sharpness    float64
}

// Sharpness measures focus as the variance of a 3×3 Laplacian over the luma
// plane. Blurry frames have a low-variance Laplacian response; crisp ones high.
func Sharpness(img image.Image) float64 {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w < 3 || h < 3 {
		return 0
	}

	// Precompute the luma plane once.
	luma := make([]float64, w*h)
	for y := range h {
		for x := range w {
			r, g, bl, _ := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
			// RGBA() returns 16-bit; scale to 0..255.
			luma[y*w+x] = (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(bl)) / 257
		}
	}

	// Laplacian response over interior pixels; accumulate mean/variance online.
	var n int
	var mean, m2 float64
	for y := 1; y < h-1; y++ {
		for x := 1; x < w-1; x++ {
			c := luma[y*w+x]
			lap := luma[(y-1)*w+x] + luma[(y+1)*w+x] + luma[y*w+x-1] + luma[y*w+x+1] - 4*c
			n++
			delta := lap - mean
			mean += delta / float64(n)
			m2 += delta * (lap - mean)
		}
	}
	if n == 0 {
		return 0
	}
	return m2 / float64(n)
}
