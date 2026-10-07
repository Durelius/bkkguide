package media

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"testing"
)

func process(t *testing.T, data []byte) Processed {
	t.Helper()
	if !Available() {
		t.Skip("ffmpeg not installed")
	}
	out, err := Process(context.Background(), data)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestRejectsNonImages(t *testing.T) {
	if !Available() {
		t.Skip("ffmpeg not installed")
	}
	if _, err := Process(context.Background(), []byte("hello")); err != ErrNotImage {
		t.Fatalf("err = %v, want ErrNotImage", err)
	}
}

func TestResizesToBothSizes(t *testing.T) {
	var buf bytes.Buffer
	png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 2000, 1000)))
	out := process(t, buf.Bytes())
	if out.Width != 1600 || out.Height != 800 {
		t.Errorf("large = %d×%d, want 1600×800", out.Width, out.Height)
	}
	if cfg, _ := jpeg.DecodeConfig(bytes.NewReader(out.Thumb)); cfg.Width != 640 || cfg.Height != 320 {
		t.Errorf("thumb = %d×%d, want 640×320", cfg.Width, cfg.Height)
	}
}

func TestPortraitBoundByLongEdge(t *testing.T) {
	var buf bytes.Buffer
	png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 3024, 4032)))
	out := process(t, buf.Bytes())
	if out.Width != 1200 || out.Height != 1600 {
		t.Errorf("large = %d×%d, want 1200×1600", out.Width, out.Height)
	}
}

// Both fixtures are a 400×200 image (red left, blue right) stored rotated:
// rot.heic via the HEIC rotation box, rot6.jpg via EXIF orientation 6.
// Upright, they are 200×400 with red on top.
func TestAppliesRotation(t *testing.T) {
	for _, name := range []string{"testdata/rot.heic", "testdata/rot6.jpg"} {
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(name)
			if err != nil {
				t.Fatal(err)
			}
			out := process(t, data)
			if out.Width != 200 || out.Height != 400 {
				t.Fatalf("size = %d×%d, want 200×400", out.Width, out.Height)
			}
			img, err := jpeg.Decode(bytes.NewReader(out.Large))
			if err != nil {
				t.Fatal(err)
			}
			if !reddish(img.At(100, 20)) || reddish(img.At(100, 380)) {
				t.Errorf("not upright: top %v, bottom %v", img.At(100, 20), img.At(100, 380))
			}
		})
	}
}

func reddish(c color.Color) bool {
	r, g, b, _ := c.RGBA()
	return r>>8 > 180 && g>>8 < 90 && b>>8 < 90
}
