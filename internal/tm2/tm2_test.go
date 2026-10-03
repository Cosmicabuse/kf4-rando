package tm2

import (
	"encoding/binary"
	"image"
	"image/color"
	"testing"
)

// buildTM2 hand-builds a minimal PSM-0 TIM2 file: a width x height image
// where every pixel is the same BGRA + half-alpha byte quad.
func buildTM2(width, height int, b, g, r, halfAlpha byte) []byte {
	pixelSize := width * height * 4
	buf := make([]byte, headerSize+pixelSize)

	binary.LittleEndian.PutUint32(buf[0:4], uint32(len(buf))) // tm2Length
	// clutOffset, unknown0x08, unknown0x0c stay 0

	logW := log2(width)
	logH := log2(height)
	var gsTexReg uint64
	gsTexReg |= uint64(psm0) << 20
	gsTexReg |= uint64(logW) << 26
	gsTexReg |= uint64(logH) << 30
	binary.LittleEndian.PutUint64(buf[0x10:0x18], gsTexReg)

	for i := 0; i < pixelSize; i += 4 {
		buf[headerSize+i] = b
		buf[headerSize+i+1] = g
		buf[headerSize+i+2] = r
		buf[headerSize+i+3] = halfAlpha
	}
	return buf
}

func log2(n int) int {
	l := 0
	for n > 1 {
		n >>= 1
		l++
	}
	return l
}

func TestDecode_HeaderFields(t *testing.T) {
	data := buildTM2(4, 8, 10, 20, 30, 64)
	img, header, err := Decode(data)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if header.width() != 4 || header.height() != 8 {
		t.Errorf("header dims = %dx%d, want 4x8", header.width(), header.height())
	}
	if b := img.Bounds(); b.Dx() != 4 || b.Dy() != 8 {
		t.Errorf("image bounds = %v, want 4x8", b)
	}
}

func TestDecode_PixelValues(t *testing.T) {
	// BGRA=10,20,30, halfAlpha=64 -> RGBA should be R=30,G=20,B=10,A=min(128,255)=128
	data := buildTM2(2, 2, 10, 20, 30, 64)
	img, _, err := Decode(data)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	r, g, b, a := img.NRGBAAt(0, 0).R, img.NRGBAAt(0, 0).G, img.NRGBAAt(0, 0).B, img.NRGBAAt(0, 0).A
	if r != 30 || g != 20 || b != 10 || a != 128 {
		t.Errorf("pixel = R%d G%d B%d A%d, want R30 G20 B10 A128", r, g, b, a)
	}
}

func TestDecode_AlphaClampsAt255(t *testing.T) {
	// halfAlpha=200 -> 200*2=400, clamped to 255
	data := buildTM2(2, 2, 0, 0, 0, 200)
	img, _, err := Decode(data)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if a := img.NRGBAAt(0, 0).A; a != 255 {
		t.Errorf("alpha = %d, want 255 (clamped)", a)
	}
}

func TestDecode_RejectsUnsupportedPSM(t *testing.T) {
	data := buildTM2(2, 2, 0, 0, 0, 0)
	// force PSM to 19 (8bpp indexed) in the gsTexReg field
	gsTexReg := binary.LittleEndian.Uint64(data[0x10:0x18])
	gsTexReg = (gsTexReg &^ (0x3F << 20)) | (19 << 20)
	binary.LittleEndian.PutUint64(data[0x10:0x18], gsTexReg)

	if _, _, err := Decode(data); err == nil {
		t.Error("expected an error decoding a non-PSM-0 file, got nil")
	}
}

func TestDecode_RejectsTooShort(t *testing.T) {
	if _, _, err := Decode([]byte{1, 2, 3}); err == nil {
		t.Error("expected an error decoding too-short data, got nil")
	}
}

func TestEncodeDecode_RoundTrip(t *testing.T) {
	orig := buildTM2(4, 4, 10, 20, 30, 64)
	img, header, err := Decode(orig)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	reencoded, err := Encode(header, img)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if len(reencoded) != len(orig) {
		t.Fatalf("re-encoded length = %d, want %d", len(reencoded), len(orig))
	}
	img2, _, err := Decode(reencoded)
	if err != nil {
		t.Fatalf("Decode(re-encoded): %v", err)
	}
	if img2.NRGBAAt(0, 0) != img.NRGBAAt(0, 0) {
		t.Errorf("round-tripped pixel = %v, want %v", img2.NRGBAAt(0, 0), img.NRGBAAt(0, 0))
	}
}

func TestEncode_RejectsDimensionMismatch(t *testing.T) {
	_, header, err := Decode(buildTM2(4, 4, 0, 0, 0, 0))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	wrongSize := buildTM2(2, 2, 0, 0, 0, 0)
	wrongImg, _, err := Decode(wrongSize)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if _, err := Encode(header, wrongImg); err == nil {
		t.Error("expected an error encoding a dimension mismatch, got nil")
	}
}

func solidNRGBA(w, h int, c color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

func TestResize_OutputDimensions(t *testing.T) {
	src := solidNRGBA(10, 10, color.NRGBA{R: 100, G: 100, B: 100, A: 255})
	out := Resize(src, 6, 4)
	if b := out.Bounds(); b.Dx() != 6 || b.Dy() != 4 {
		t.Errorf("output bounds = %v, want 6x4", b)
	}
}

func TestResize_SolidColorStaysConstant(t *testing.T) {
	want := color.NRGBA{R: 50, G: 100, B: 150, A: 255}
	src := solidNRGBA(8, 8, want)
	out := Resize(src, 20, 20)
	// interior pixels should match exactly for a fully-opaque solid fill;
	// only check a point away from any edge-clamping ambiguity
	got := out.NRGBAAt(10, 10)
	if got != want {
		t.Errorf("resized solid color = %v, want %v", got, want)
	}
}

func TestResize_ZeroDimensionsDoesNotPanic(t *testing.T) {
	src := solidNRGBA(4, 4, color.NRGBA{A: 255})
	out := Resize(src, 0, 0)
	if b := out.Bounds(); b.Dx() != 0 || b.Dy() != 0 {
		t.Errorf("expected an empty image, got %v", b)
	}
}

func TestComposite_OpaqueOverwrites(t *testing.T) {
	dst := solidNRGBA(4, 4, color.NRGBA{R: 0, G: 0, B: 255, A: 255})
	src := solidNRGBA(2, 2, color.NRGBA{R: 255, G: 0, B: 0, A: 255})
	Composite(dst, src, 1, 1)

	if got := dst.NRGBAAt(1, 1); got != (color.NRGBA{R: 255, G: 0, B: 0, A: 255}) {
		t.Errorf("pixel under opaque src = %v, want opaque red", got)
	}
	if got := dst.NRGBAAt(0, 0); got != (color.NRGBA{R: 0, G: 0, B: 255, A: 255}) {
		t.Errorf("pixel outside src = %v, want unchanged blue", got)
	}
}

func TestComposite_TransparentSrcLeavesDestUnchanged(t *testing.T) {
	dst := solidNRGBA(4, 4, color.NRGBA{R: 0, G: 0, B: 255, A: 255})
	src := solidNRGBA(2, 2, color.NRGBA{R: 255, G: 0, B: 0, A: 0})
	Composite(dst, src, 1, 1)

	if got := dst.NRGBAAt(1, 1); got != (color.NRGBA{R: 0, G: 0, B: 255, A: 255}) {
		t.Errorf("pixel under fully-transparent src = %v, want unchanged blue", got)
	}
}

func TestComposite_ClipsOutOfBounds(t *testing.T) {
	dst := solidNRGBA(4, 4, color.NRGBA{A: 255})
	src := solidNRGBA(4, 4, color.NRGBA{R: 255, A: 255})
	// should not panic even though src extends past dst's bounds
	Composite(dst, src, 2, 2)
	if got := dst.NRGBAAt(2, 2); got.R != 255 {
		t.Errorf("in-bounds overlap pixel R = %d, want 255", got.R)
	}
}
