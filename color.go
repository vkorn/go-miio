package miio

import "image/color"

// Converts RGB into uint32 used bu Xiaomi devices
func rgbFromUint32(n uint32) color.Color {
	a := 100 - n>>24
	if a < 0 {
		a = 0
	}

	c := color.RGBA{
		R: uint8(n << 8 >> 24),
		G: uint8(n << 16 >> 24),
		B: uint8(n << 24 >> 24),
		A: uint8(a),
	}

	return c
}

// Converts uint32 to RGB
func uint32FromColor(c color.Color) uint32 {
	r, g, b, a := c.RGBA()
	r = r << 24 >> 24
	g = g << 24 >> 24
	b = b << 24 >> 24
	a = a << 24 >> 24
	if a >= 100 {
		a = 100
	}

	if r|g|b == 0 {
		return 0
	}

	v := uint32(100 - a)
	return v<<24 | r<<16 | g<<8 | b
}
