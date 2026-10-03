// Package titlelogo adds "Randomized" to the title logo. The logo is two
// halves with no spare pixels, so both shrink the same amount and the text
// goes in the space freed under the right half.
package titlelogo

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"math"

	"kf4randomizer/internal/tm2"
)

const scale = 0.78

// tuned by eye on a real title screen
const (
	rightMarginFull  = 247 // logo2's right content margin at full size
	overlayLeftNudge = 8
	overlayUpNudge   = 15
	overlayHeightPct = 0.72
)

// ApplyBranding returns both halves re-encoded at their original file size.
func ApplyBranding(logo1TM2, logo2TM2, overlayPNG []byte) (newLogo1TM2, newLogo2TM2 []byte, err error) {
	logo1, header1, err := tm2.Decode(logo1TM2)
	if err != nil {
		return nil, nil, fmt.Errorf("titlelogo: decoding logo1: %w", err)
	}
	logo2, header2, err := tm2.Decode(logo2TM2)
	if err != nil {
		return nil, nil, fmt.Errorf("titlelogo: decoding logo2: %w", err)
	}
	if logo1.Bounds() != logo2.Bounds() {
		return nil, nil, fmt.Errorf("titlelogo: logo1 is %v, logo2 is %v, expected matching dimensions", logo1.Bounds(), logo2.Bounds())
	}
	w, h := logo1.Bounds().Dx(), logo1.Bounds().Dy()
	newSize := int(float64(w) * scale) // logo1/logo2 are square, so one scale covers both axes

	overlayImg, err := png.Decode(bytes.NewReader(overlayPNG))
	if err != nil {
		return nil, nil, fmt.Errorf("titlelogo: decoding overlay PNG: %w", err)
	}
	overlay := toNRGBA(overlayImg)

	shrunk1 := tm2.Resize(logo1, newSize, newSize)
	canvas1 := image.NewNRGBA(image.Rect(0, 0, w, h))
	// anchor top-right so logo1 stays flush against logo2's left edge.
	tm2.Composite(canvas1, shrunk1, w-newSize, 0)

	shrunk2 := tm2.Resize(logo2, newSize, newSize)
	canvas2 := image.NewNRGBA(image.Rect(0, 0, w, h))
	// anchor top-left to match logo1's right edge at the shared seam.
	tm2.Composite(canvas2, shrunk2, 0, 0)

	freedHeight := h - newSize
	targetOverlayHeight := int(float64(freedHeight) * overlayHeightPct)
	ob := overlay.Bounds()
	overlayScale := float64(targetOverlayHeight) / float64(ob.Dy())
	overlayResized := tm2.Resize(overlay, int(float64(ob.Dx())*overlayScale), targetOverlayHeight)

	rightMarginScaled := int(math.Round(float64(rightMarginFull) * scale))
	ox := rightMarginScaled - overlayResized.Bounds().Dx() - overlayLeftNudge
	oy := newSize + (freedHeight-overlayResized.Bounds().Dy())/2 - overlayUpNudge
	tm2.Composite(canvas2, overlayResized, ox, oy)

	newLogo1TM2, err = tm2.Encode(header1, canvas1)
	if err != nil {
		return nil, nil, fmt.Errorf("titlelogo: encoding logo1: %w", err)
	}
	newLogo2TM2, err = tm2.Encode(header2, canvas2)
	if err != nil {
		return nil, nil, fmt.Errorf("titlelogo: encoding logo2: %w", err)
	}
	return newLogo1TM2, newLogo2TM2, nil
}

func toNRGBA(src image.Image) *image.NRGBA {
	if n, ok := src.(*image.NRGBA); ok {
		return n
	}
	b := src.Bounds()
	dst := image.NewNRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			dst.Set(x, y, src.At(x, y))
		}
	}
	return dst
}
