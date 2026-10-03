package titlelogo

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"testing"

	"kf4randomizer/internal/tm2"
)

// buildTestTM2 hand-builds a minimal square PSM-0 TIM2 file.
func buildTestTM2(size int, b, g, r, halfAlpha byte) []byte {
	const headerSize = 0x20
	const psm0 = 0
	pixelSize := size * size * 4
	buf := make([]byte, headerSize+pixelSize)
	binary.LittleEndian.PutUint32(buf[0:4], uint32(len(buf)))

	log2 := 0
	for n := size; n > 1; n >>= 1 {
		log2++
	}
	var gsTexReg uint64
	gsTexReg |= uint64(psm0) << 20
	gsTexReg |= uint64(log2) << 26
	gsTexReg |= uint64(log2) << 30
	binary.LittleEndian.PutUint64(buf[0x10:0x18], gsTexReg)

	for i := 0; i < pixelSize; i += 4 {
		buf[headerSize+i], buf[headerSize+i+1], buf[headerSize+i+2], buf[headerSize+i+3] = b, g, r, halfAlpha
	}
	return buf
}

func buildTestOverlayPNG(w, h int) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 0, G: 200, B: 200, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err) // test setup only
	}
	return buf.Bytes()
}

func TestApplyBranding_ProducesValidSameSizeTM2s(t *testing.T) {
	// The layout constants assume 256x256.
	logo1 := buildTestTM2(256, 100, 100, 100, 128) // opaque gray
	logo2 := buildTestTM2(256, 50, 50, 50, 128)
	overlay := buildTestOverlayPNG(8, 4)

	newLogo1, newLogo2, err := ApplyBranding(logo1, logo2, overlay)
	if err != nil {
		t.Fatalf("ApplyBranding: %v", err)
	}

	if len(newLogo1) != len(logo1) {
		t.Errorf("logo1: got %d bytes, want %d (the DAT slot is a fixed size)", len(newLogo1), len(logo1))
	}
	if len(newLogo2) != len(logo2) {
		t.Errorf("logo2: got %d bytes, want %d (the DAT slot is a fixed size)", len(newLogo2), len(logo2))
	}

	img1, _, err := tm2.Decode(newLogo1)
	if err != nil {
		t.Fatalf("decoding new logo1: %v", err)
	}
	if b := img1.Bounds(); b.Dx() != 256 || b.Dy() != 256 {
		t.Errorf("new logo1 dims = %v, want 256x256", b)
	}

	img2, _, err := tm2.Decode(newLogo2)
	if err != nil {
		t.Fatalf("decoding new logo2: %v", err)
	}
	if b := img2.Bounds(); b.Dx() != 256 || b.Dy() != 256 {
		t.Errorf("new logo2 dims = %v, want 256x256", b)
	}
}

func TestApplyBranding_OverlayPixelsAppearInLogo2(t *testing.T) {
	logo1 := buildTestTM2(256, 0, 0, 0, 0) // fully transparent, won't obscure anything
	logo2 := buildTestTM2(256, 0, 0, 0, 0)
	overlay := buildTestOverlayPNG(8, 4)

	_, newLogo2, err := ApplyBranding(logo1, logo2, overlay)
	if err != nil {
		t.Fatalf("ApplyBranding: %v", err)
	}
	img2, _, err := tm2.Decode(newLogo2)
	if err != nil {
		t.Fatalf("decoding new logo2: %v", err)
	}

	found := false
	b := img2.Bounds()
	for y := b.Min.Y; y < b.Max.Y && !found; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if p := img2.NRGBAAt(x, y); p.G > 100 && p.B > 100 && p.R < 50 {
				found = true
				break
			}
		}
	}
	if !found {
		t.Error("expected to find the overlay's cyan color somewhere in the branded logo2, found none")
	}
}

func TestApplyBranding_RejectsDimensionMismatch(t *testing.T) {
	logo1 := buildTestTM2(32, 0, 0, 0, 0)
	logo2 := buildTestTM2(16, 0, 0, 0, 0)
	overlay := buildTestOverlayPNG(8, 4)

	if _, _, err := ApplyBranding(logo1, logo2, overlay); err == nil {
		t.Error("expected an error for mismatched logo1/logo2 dimensions, got nil")
	}
}

func TestApplyBranding_RejectsInvalidOverlay(t *testing.T) {
	logo1 := buildTestTM2(32, 0, 0, 0, 0)
	logo2 := buildTestTM2(32, 0, 0, 0, 0)

	if _, _, err := ApplyBranding(logo1, logo2, []byte("not a png")); err == nil {
		t.Error("expected an error for invalid overlay PNG bytes, got nil")
	}
}

func TestApplyBranding_RejectsInvalidTM2(t *testing.T) {
	overlay := buildTestOverlayPNG(8, 4)
	if _, _, err := ApplyBranding([]byte("not a tm2"), buildTestTM2(32, 0, 0, 0, 0), overlay); err == nil {
		t.Error("expected an error for invalid logo1 bytes, got nil")
	}
}
