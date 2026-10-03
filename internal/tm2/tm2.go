// Package tm2 reads and writes TIM2 textures, 32bpp (PSM 0) only. Layout
// from ToolsForKFIV's TM2.cs.
package tm2

import (
	"encoding/binary"
	"fmt"
	"image"
)

const headerSize = 0x20

// psm0 is the only pixel storage mode this package handles: 32bpp ARGB,
// stored BGRA with alpha halved (PS2 GS convention).
const psm0 = 0

// Header is a TIM2 file's 32-byte header, kept as is so Encode can
// reuse it as a template for a same-dimensions edit.
type Header [headerSize]byte

func (h Header) clutOffset() uint32 { return binary.LittleEndian.Uint32(h[4:8]) }
func (h Header) gsTexReg() uint64   { return binary.LittleEndian.Uint64(h[0x10:0x18]) }
func (h Header) psm() uint64        { return (h.gsTexReg() >> 20) & 0x3F }
func (h Header) width() int         { return 1 << ((h.gsTexReg() >> 26) & 0xF) }
func (h Header) height() int        { return 1 << ((h.gsTexReg() >> 30) & 0xF) }

// Decode returns the image and the header to hand back to Encode.
func Decode(data []byte) (*image.NRGBA, Header, error) {
	var h Header
	if len(data) < headerSize {
		return nil, h, fmt.Errorf("tm2: %d bytes is too short for a %d-byte header", len(data), headerSize)
	}
	copy(h[:], data[:headerSize])

	if h.clutOffset() != 0 {
		return nil, h, fmt.Errorf("tm2: clut-based textures aren't supported (clutOffset=%d)", h.clutOffset())
	}
	if h.psm() != psm0 {
		return nil, h, fmt.Errorf("tm2: only PSM 0 (32bpp ARGB) is supported, got PSM %d", h.psm())
	}

	w, ht := h.width(), h.height()
	pixelSize := w * ht * 4
	if len(data) < headerSize+pixelSize {
		return nil, h, fmt.Errorf("tm2: %dx%d PSM 0 needs %d pixel bytes, file only has %d", w, ht, pixelSize, len(data)-headerSize)
	}

	img := image.NewNRGBA(image.Rect(0, 0, w, ht))
	for i := 0; i < pixelSize; i += 4 {
		b, g, r, a := data[headerSize+i], data[headerSize+i+1], data[headerSize+i+2], data[headerSize+i+3]
		img.Pix[i] = r
		img.Pix[i+1] = g
		img.Pix[i+2] = b
		img.Pix[i+3] = byte(min(int(a)*2, 255))
	}
	return img, h, nil
}

// Encode writes img back out as a TIM2 file, reusing header as a template.
// header must come from Decode-ing a same-dimensions, PSM-0 source file.
func Encode(header Header, img *image.NRGBA) ([]byte, error) {
	w, ht := header.width(), header.height()
	if b := img.Bounds(); b.Dx() != w || b.Dy() != ht {
		return nil, fmt.Errorf("tm2: image is %dx%d, header expects %dx%d", b.Dx(), b.Dy(), w, ht)
	}

	pixelSize := w * ht * 4
	out := make([]byte, headerSize+pixelSize)
	copy(out, header[:])
	binary.LittleEndian.PutUint32(out, uint32(pixelSize+headerSize))

	for i := 0; i < pixelSize; i += 4 {
		r, g, b, a := img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3]
		out[headerSize+i] = b
		out[headerSize+i+1] = g
		out[headerSize+i+2] = r
		out[headerSize+i+3] = byte(min((int(a)+1)/2, 128))
	}
	return out, nil
}

// Resize scales img to newWidth x newHeight using bilinear interpolation,
// premultiplying alpha before interpolating to avoid dark/light fringing.
func Resize(img *image.NRGBA, newWidth, newHeight int) *image.NRGBA {
	src := img.Bounds()
	sw, sh := src.Dx(), src.Dy()
	out := image.NewNRGBA(image.Rect(0, 0, newWidth, newHeight))

	if newWidth <= 0 || newHeight <= 0 || sw == 0 || sh == 0 {
		return out
	}

	// premultiply once up front, as float32 for interpolation precision
	type pixel struct{ r, g, b, a float32 }
	pm := make([]pixel, sw*sh)
	for y := 0; y < sh; y++ {
		for x := 0; x < sw; x++ {
			i := img.PixOffset(src.Min.X+x, src.Min.Y+y)
			a := float32(img.Pix[i+3]) / 255
			pm[y*sw+x] = pixel{
				r: float32(img.Pix[i]) * a,
				g: float32(img.Pix[i+1]) * a,
				b: float32(img.Pix[i+2]) * a,
				a: a,
			}
		}
	}
	at := func(x, y int) pixel {
		if x < 0 {
			x = 0
		}
		if x >= sw {
			x = sw - 1
		}
		if y < 0 {
			y = 0
		}
		if y >= sh {
			y = sh - 1
		}
		return pm[y*sw+x]
	}

	xScale := float32(sw) / float32(newWidth)
	yScale := float32(sh) / float32(newHeight)
	for oy := 0; oy < newHeight; oy++ {
		sy := (float32(oy)+0.5)*yScale - 0.5
		y0 := int(sy)
		if float32(y0) > sy {
			y0--
		}
		fy := sy - float32(y0)

		for ox := 0; ox < newWidth; ox++ {
			sx := (float32(ox)+0.5)*xScale - 0.5
			x0 := int(sx)
			if float32(x0) > sx {
				x0--
			}
			fx := sx - float32(x0)

			p00, p10 := at(x0, y0), at(x0+1, y0)
			p01, p11 := at(x0, y0+1), at(x0+1, y0+1)

			lerp := func(a, b, t float32) float32 { return a + (b-a)*t }
			r := lerp(lerp(p00.r, p10.r, fx), lerp(p01.r, p11.r, fx), fy)
			g := lerp(lerp(p00.g, p10.g, fx), lerp(p01.g, p11.g, fx), fy)
			b := lerp(lerp(p00.b, p10.b, fx), lerp(p01.b, p11.b, fx), fy)
			a := lerp(lerp(p00.a, p10.a, fx), lerp(p01.a, p11.a, fx), fy)

			oi := out.PixOffset(ox, oy)
			if a > 0 {
				out.Pix[oi] = clamp255(r / a)
				out.Pix[oi+1] = clamp255(g / a)
				out.Pix[oi+2] = clamp255(b / a)
			}
			out.Pix[oi+3] = clamp255(a * 255)
		}
	}
	return out
}

func clamp255(v float32) byte {
	if v <= 0 {
		return 0
	}
	if v >= 255 {
		return 255
	}
	return byte(v + 0.5)
}

// Composite alpha-blends src onto dst at (x, y), clipping at dst's edges.
func Composite(dst *image.NRGBA, src *image.NRGBA, x, y int) {
	sb := src.Bounds()
	db := dst.Bounds()
	for sy := 0; sy < sb.Dy(); sy++ {
		dy := y + sy
		if dy < db.Min.Y || dy >= db.Max.Y {
			continue
		}
		for sx := 0; sx < sb.Dx(); sx++ {
			dx := x + sx
			if dx < db.Min.X || dx >= db.Max.X {
				continue
			}
			si := src.PixOffset(sb.Min.X+sx, sb.Min.Y+sy)
			sa := float32(src.Pix[si+3]) / 255
			if sa <= 0 {
				continue
			}
			di := dst.PixOffset(dx, dy)
			da := float32(dst.Pix[di+3]) / 255
			outA := sa + da*(1-sa)
			blend := func(sc, dc byte) byte {
				if outA <= 0 {
					return 0
				}
				v := (float32(sc)*sa + float32(dc)*da*(1-sa)) / outA
				return clamp255(v)
			}
			dst.Pix[di] = blend(src.Pix[si], dst.Pix[di])
			dst.Pix[di+1] = blend(src.Pix[si+1], dst.Pix[di+1])
			dst.Pix[di+2] = blend(src.Pix[si+2], dst.Pix[di+2])
			dst.Pix[di+3] = clamp255(outA * 255)
		}
	}
}
