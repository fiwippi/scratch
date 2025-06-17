package quantise

import (
	"image"
	"image/color"
	"image/draw"
)

type Ditherer interface {
	Dither(p color.Palette, img image.Image) image.Image
}

var (
	_ Ditherer = None{}
	_ Ditherer = FloydSteinberg{}
)

type None struct{}

func (_ None) Dither(p color.Palette, img image.Image) image.Image {
	bounds := img.Bounds()
	dst := image.NewPaletted(bounds, p)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := img.At(x, y)
			dst.Set(x, y, p.Convert(c))
		}
	}

	return dst
}

type FloydSteinberg struct{}

func (_ FloydSteinberg) Dither(p color.Palette, img image.Image) image.Image {
	bounds := img.Bounds()
	dst := image.NewPaletted(bounds, p)
	draw.FloydSteinberg.Draw(dst, bounds, img, image.Point{})
	return dst
}
