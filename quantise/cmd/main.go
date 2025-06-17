package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	"image/png"
	"os"
	"os/exec"
	"quantise"
	"strconv"
)

func run(in, out string, m int, dither bool, framerate int) error {
	f, err := os.Open(in)
	if err != nil {
		return err
	}
	img, _, err := image.Decode(f)
	if err != nil {
		return err
	}

	var cmdArgs []string
	cmdArgs = append(cmdArgs,
		"-hide_banner", "-loglevel", "error",
		"-f", "image2pipe", "-c:v", "png", "-r", strconv.Itoa(framerate), "-i", "-",
		"-y", "-an", "-pix_fmt", "yuv420p",
	)
	cmdArgs = append(cmdArgs, out)
	cmd := exec.Command("ffmpeg", cmdArgs...)
	var e bytes.Buffer
	cmd.Stderr = &e

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	defer stdin.Close()
	if err := cmd.Start(); err != nil {
		return err
	}

	for i := 2; i <= m; i++ {
		p := quantise.Quantise(img, i)
		d := quantise.Ditherer(quantise.None{})
		if dither {
			d = quantise.FloydSteinberg{}
		}

		top := d.Dither(p, img)
		bottom := quantise.Palette(p, top.Bounds().Dx()/len(p))

		width := top.Bounds().Dx()
		height := top.Bounds().Dy() + bottom.Bounds().Dy()
		// In case we are encoding mp4, we always make sure the
		// width and height are divisible by two
		if width%2 != 0 {
			width += 1
		}
		if height%2 != 0 {
			height += 1
		}

		frame := image.NewRGBA(image.Rect(0, 0, width, height))
		offset := (width - bottom.Bounds().Dx()) / 2
		bottomRect := image.Rect(offset, top.Bounds().Dy(), offset+bottom.Bounds().Dx(), height)
		draw.Draw(frame, frame.Bounds(), &image.Uniform{color.Black}, image.Point{}, draw.Src)
		draw.Draw(frame, top.Bounds(), top, image.Point{}, draw.Over)
		draw.Draw(frame, bottomRect, bottom, image.Point{}, draw.Over)

		if err := png.Encode(stdin, frame); err != nil {
			return err
		}
	}

	stdin.Close() // Tells FFMPEG to stop
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("ffmpeg: %w:\n%s", err, e.String())
	}

	return nil
}

func main() {
	colours := flag.Int("colours", 16, "")
	dither := flag.Bool("dither", false, "")
	framerate := flag.Int("framerate", 2, "")
	flag.Parse()

	in := flag.Arg(0)
	if in == "" {
		fmt.Println("Input image not specified")
		os.Exit(1)
	}
	out := flag.Arg(1)
	if out == "" {
		fmt.Println("Output image not specified")
		os.Exit(1)
	}

	if err := run(in, out, *colours, *dither, *framerate); err != nil {
		fmt.Println("Failed:", err)
	}
}
