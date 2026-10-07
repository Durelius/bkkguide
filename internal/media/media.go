// Package media turns uploaded photos into compressed JPEGs with ffmpeg.
//
// Any format ffmpeg reads is accepted (JPEG, PNG, WebP, HEIC from iPhones).
// ffmpeg first decodes the upload to raw pixels, applying EXIF/HEIC rotation
// and dropping metadata (GPS included), then encodes one JPEG per size. The
// original upload is never stored.
package media

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image/jpeg"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const (
	MaxUploadBytes = 25 << 20
	maxPixels      = 50_000_000
	timeout        = 60 * time.Second
)

// Size is one stored rendition: max long edge and ffmpeg JPEG quality (2 best – 31 worst).
type Size struct {
	Edge    int
	Quality int
}

var (
	Large = Size{Edge: 1600, Quality: 3}
	Thumb = Size{Edge: 640, Quality: 4}
)

var ErrNotImage = errors.New("that file isn't a photo we can read")

// Processed holds the compressed JPEGs for one photo.
type Processed struct {
	Thumb, Large  []byte
	Width, Height int // of the large version
}

// Available reports whether ffmpeg is on PATH.
func Available() bool {
	_, err := exec.LookPath("ffmpeg")
	return err == nil
}

func Process(ctx context.Context, data []byte) (Processed, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// HEIC needs a seekable input, so go through a temp file.
	in, err := os.CreateTemp("", "bkk-upload-*")
	if err != nil {
		return Processed{}, err
	}
	defer os.Remove(in.Name())
	if _, err := in.Write(data); err != nil {
		in.Close()
		return Processed{}, err
	}
	in.Close()

	raw, err := ffmpeg(ctx, nil, "-i", in.Name(), "-frames:v", "1", "-map_metadata", "-1", "-f", "image2pipe", "-c:v", "ppm", "pipe:1")
	if err != nil {
		if ctx.Err() != nil {
			return Processed{}, errors.New("processing the photo took too long")
		}
		return Processed{}, ErrNotImage
	}
	w, h, err := ppmSize(raw)
	if err != nil {
		return Processed{}, ErrNotImage
	}
	if w*h > maxPixels {
		return Processed{}, fmt.Errorf("that image is too large (%d×%d)", w, h)
	}

	var out Processed
	if out.Large, err = encode(ctx, raw, Large); err != nil {
		return Processed{}, err
	}
	if out.Thumb, err = encode(ctx, raw, Thumb); err != nil {
		return Processed{}, err
	}
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(out.Large))
	if err != nil {
		return Processed{}, err
	}
	out.Width, out.Height = cfg.Width, cfg.Height
	return out, nil
}

// encode scales raw PPM pixels so the long edge is at most s.Edge (never up)
// and compresses to JPEG.
func encode(ctx context.Context, raw []byte, s Size) ([]byte, error) {
	scale := fmt.Sprintf("scale=w='if(gte(iw,ih),min(%[1]d,iw),-2)':h='if(gte(iw,ih),-2,min(%[1]d,ih))':flags=lanczos,format=yuvj420p", s.Edge)
	return ffmpeg(ctx, raw, "-f", "ppm_pipe", "-i", "pipe:0", "-vf", scale,
		"-q:v", strconv.Itoa(s.Quality), "-frames:v", "1", "-f", "mjpeg", "pipe:1")
}

func ffmpeg(ctx context.Context, stdin []byte, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "ffmpeg", append([]string{"-nostdin", "-hide_banner", "-v", "error"}, args...)...)
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
		cmd.Args = append(cmd.Args[:1], cmd.Args[2:]...) // -nostdin would stop us feeding pipe:0
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg: %w: %s", err, bytes.TrimSpace(stderr.Bytes()))
	}
	if stdout.Len() == 0 {
		return nil, errors.New("ffmpeg produced no output")
	}
	return stdout.Bytes(), nil
}

// ppmSize reads width and height from a binary PPM header ("P6\n<w> <h>\n<max>\n").
func ppmSize(b []byte) (int, int, error) {
	f := strings.Fields(string(b[:min(len(b), 64)]))
	if len(f) < 4 || f[0] != "P6" {
		return 0, 0, errors.New("not a PPM")
	}
	w, err1 := strconv.Atoi(f[1])
	h, err2 := strconv.Atoi(f[2])
	if err1 != nil || err2 != nil || w <= 0 || h <= 0 {
		return 0, 0, errors.New("bad PPM size")
	}
	return w, h, nil
}
