package console

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"path/filepath"
	"slices"
	"strings"
)

const (
	maxImageSize      = 32 << 20
	maxImagePixels    = 512
	hexBytesPerLine   = 16
	hexGroupSize      = 8
	firstPrintable    = 0x20
	lastPrintable     = 0x7e
	colorChannelShift = 8
)

var imageExtensions = []string{".png", ".jpg", ".jpeg", ".gif"}

type previewKind int

const (
	textPreview previewKind = iota
	imagePreview
	binaryPreview
)

type previewContent struct {
	kind   previewKind
	data   []byte
	image  image.Image
	format string
	width  int
	height int
}

func previewLimit(name string) int64 {
	if slices.Contains(imageExtensions, strings.ToLower(filepath.Ext(name))) {
		return maxImageSize
	}

	return maxPreviewSize
}

func classifyContent(content []byte) previewContent {
	if isText(content) {
		return previewContent{kind: textPreview, data: content}
	}

	decoded, format, err := image.Decode(bytes.NewReader(content))
	if err != nil {
		return previewContent{kind: binaryPreview, data: content}
	}

	bounds := decoded.Bounds()

	return previewContent{
		kind:   imagePreview,
		image:  thumbnail(decoded, maxImagePixels),
		format: strings.ToUpper(format),
		width:  bounds.Dx(),
		height: bounds.Dy(),
	}
}

func textLines(content []byte) []string {
	text := strings.TrimSuffix(strings.ToValidUTF8(string(content), ""), "\n")
	return strings.Split(strings.ReplaceAll(text, "\t", "    "), "\n")
}

func hexLines(content []byte) []string {
	lines := make([]string, 0, (len(content)+hexBytesPerLine-1)/hexBytesPerLine)

	for offset := 0; offset < len(content); offset += hexBytesPerLine {
		chunk := content[offset:min(offset+hexBytesPerLine, len(content))]

		var hexPart, textPart strings.Builder
		for i := range hexBytesPerLine {
			if i == hexGroupSize {
				hexPart.WriteByte(' ')
			}
			if i >= len(chunk) {
				hexPart.WriteString("   ")
				continue
			}
			fmt.Fprintf(&hexPart, "%02x ", chunk[i])
			textPart.WriteByte(printable(chunk[i]))
		}

		lines = append(lines, mutedStyle.Render(fmt.Sprintf("%08x", offset))+"  "+hexPart.String()+" "+
			mutedStyle.Render("|")+textPart.String()+mutedStyle.Render("|"))
	}

	return lines
}

func printable(value byte) byte {
	if value < firstPrintable || value > lastPrintable {
		return '.'
	}

	return value
}

func thumbnail(source image.Image, maxSide int) image.Image {
	bounds := source.Bounds()
	if bounds.Dx() <= maxSide && bounds.Dy() <= maxSide {
		return source
	}

	scale := min(float64(maxSide)/float64(bounds.Dx()), float64(maxSide)/float64(bounds.Dy()))

	return resample(source, max(1, int(float64(bounds.Dx())*scale)), max(1, int(float64(bounds.Dy())*scale)))
}

func resample(source image.Image, width, height int) *image.RGBA {
	bounds := source.Bounds()
	target := image.NewRGBA(image.Rect(0, 0, width, height))

	for y := range height {
		top := bounds.Min.Y + y*bounds.Dy()/height
		bottom := max(top+1, bounds.Min.Y+(y+1)*bounds.Dy()/height)
		for x := range width {
			left := bounds.Min.X + x*bounds.Dx()/width
			right := max(left+1, bounds.Min.X+(x+1)*bounds.Dx()/width)
			target.Set(x, y, averageColor(source, image.Rect(left, top, right, bottom)))
		}
	}

	return target
}

func averageColor(source image.Image, area image.Rectangle) rgb {
	var red, green, blue, count uint64

	for y := area.Min.Y; y < area.Max.Y; y++ {
		for x := area.Min.X; x < area.Max.X; x++ {
			r, g, b, _ := source.At(x, y).RGBA()
			red += uint64(r >> colorChannelShift)
			green += uint64(g >> colorChannelShift)
			blue += uint64(b >> colorChannelShift)
			count++
		}
	}

	return rgb{r: uint8(red / count), g: uint8(green / count), b: uint8(blue / count)}
}

type rgb struct {
	r, g, b uint8
}

func (c rgb) RGBA() (r, g, b, a uint32) {
	return uint32(c.r) * 0x101, uint32(c.g) * 0x101, uint32(c.b) * 0x101, 0xffff
}

func imageLines(source image.Image, width, height int) []string {
	bounds := source.Bounds()
	if width <= 0 || height <= 0 || bounds.Empty() {
		return nil
	}

	scale := min(float64(width)/float64(bounds.Dx()), float64(height*2)/float64(bounds.Dy()))
	scaled := resample(source, max(1, int(float64(bounds.Dx())*scale)), max(1, int(float64(bounds.Dy())*scale)))
	size := scaled.Bounds()

	lines := make([]string, 0, (size.Dy()+1)/2)
	for y := 0; y < size.Dy(); y += 2 {
		var line strings.Builder
		for x := range size.Dx() {
			top := scaled.RGBAAt(x, y)
			bottom := top
			if y+1 < size.Dy() {
				bottom = scaled.RGBAAt(x, y+1)
			}
			fmt.Fprintf(&line, "\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm▀", top.R, top.G, top.B, bottom.R, bottom.G, bottom.B)
		}
		line.WriteString("\x1b[0m")
		lines = append(lines, line.String())
	}

	return lines
}
