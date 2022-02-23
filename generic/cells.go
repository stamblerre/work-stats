package generic

import (
	"image/color"
)

type Row struct {
	Cells []string
	Color color.Color
}

func paleYellow() color.Color {
	return &color.RGBA{
		R: 255,
		G: 255,
		B: 237,
	}
}
