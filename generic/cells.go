package generic

import (
	"image/color"
)

type Row struct {
	Cells    []string
	Color    color.Color
	BoldText bool
}

func paleYellow() color.Color {
	return &color.RGBA{
		R: 255,
		G: 255,
		B: 237,
	}
}

func subsubtotalGray() color.Color {
	return &color.RGBA{
		R: 247,
		G: 247,
		B: 247,
	}
}

func subtotalGray() color.Color {
	return &color.RGBA{
		R: 240,
		G: 240,
		B: 240,
	}
}

func totalGray() color.Color {
	return &color.RGBA{
		R: 232,
		G: 232,
		B: 232,
	}
}
