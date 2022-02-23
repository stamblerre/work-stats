package generic

import (
	"fmt"
	"image/color"
	"strings"
)

type Row struct {
	Cells    []*Cell
	Color    color.Color
	BoldText bool
}

type Cell struct {
	Text      string
	HyperLink string
}

func (r *Row) ToCells() []string {
	var data []string
	for _, cell := range r.Cells {
		data = append(data, cell.Text)
	}
	return data
}

func totalRow(values ...string) *Row {
	if len(values) < 1 {
		panic("empty cells added to sheet")
	}
	var cells []*Cell
	for _, text := range values {
		cells = append(cells, &Cell{Text: text})
	}
	switch strings.ToLower(cells[0].Text) {
	case "total":
		return &Row{
			Cells:    cells,
			Color:    totalGray(),
			BoldText: true,
		}
	case "subtotal":
		return &Row{
			Cells:    cells,
			Color:    subtotalGray(),
			BoldText: true,
		}
	case "":
		return &Row{
			Cells:    cells,
			Color:    subsubtotalGray(),
			BoldText: true,
		}
	default:
		panic(fmt.Sprintf("unexpected row type: %s", cells[0]))
	}
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
