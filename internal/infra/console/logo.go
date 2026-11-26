package console

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"strings"
	"sync"
)

//go:embed logo.png
var logoImage []byte

var stowmarkLogo = sync.OnceValue(func() string {
	decoded, err := png.Decode(bytes.NewReader(logoImage))
	if err != nil {
		return ""
	}

	return strings.Join(logoLines(decoded, colorSurface), "\n")
})

func logoLines(source image.Image, background color.Color) []string {
	bounds := source.Bounds()
	lines := make([]string, 0, (bounds.Dy()+1)/2)

	for y := bounds.Min.Y; y < bounds.Max.Y; y += 2 {
		var line strings.Builder
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			top := blend(source.At(x, y), background)
			bottom := background
			if y+1 < bounds.Max.Y {
				bottom = blend(source.At(x, y+1), background)
			}
			fmt.Fprintf(&line, "%s▀", halfBlockColors(top, bottom))
		}
		line.WriteString("\x1b[0m")
		lines = append(lines, line.String())
	}

	return lines
}

func halfBlockColors(top, bottom color.Color) string {
	tr, tg, tb, _ := top.RGBA()
	br, bg, bb, _ := bottom.RGBA()

	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm",
		tr>>colorChannelShift, tg>>colorChannelShift, tb>>colorChannelShift,
		br>>colorChannelShift, bg>>colorChannelShift, bb>>colorChannelShift)
}

func blend(foreground, background color.Color) color.Color {
	fr, fg, fb, fa := foreground.RGBA()
	br, bg, bb, _ := background.RGBA()
	transparency := 0xffff - fa

	return color.RGBA64{
		R: uint16(fr + br*transparency/0xffff),
		G: uint16(fg + bg*transparency/0xffff),
		B: uint16(fb + bb*transparency/0xffff),
		A: 0xffff,
	}
}
