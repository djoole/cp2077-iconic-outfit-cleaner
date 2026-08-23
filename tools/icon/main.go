package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
)

var (
	yellow = color.RGBA{R: 245, G: 230, B: 66, A: 255}
	ink    = color.RGBA{R: 7, G: 16, B: 17, A: 255}
)

func main() {
	canvas := image.NewRGBA(image.Rect(0, 0, 256, 256))
	fillPolygon(canvas, []image.Point{{16, 16}, {240, 16}, {240, 216}, {216, 240}, {16, 240}, {16, 40}}, yellow)
	drawSpark(canvas, 102, 126, 62, 24, ink)
	drawSpark(canvas, 174, 72, 28, 11, ink)
	drawSpark(canvas, 184, 178, 23, 9, ink)

	pngPath := filepath.Join("build", "appicon.png")
	icoPath := filepath.Join("build", "windows", "icon.ico")
	_ = os.MkdirAll(filepath.Dir(icoPath), 0o755)

	var pngData bytes.Buffer
	if err := png.Encode(&pngData, canvas); err != nil {
		panic(err)
	}
	if err := os.WriteFile(pngPath, pngData.Bytes(), 0o644); err != nil {
		panic(err)
	}
	if err := writeICO(icoPath, pngData.Bytes()); err != nil {
		panic(err)
	}
}

func drawSpark(dst *image.RGBA, cx, cy, longRadius, shortRadius int, value color.RGBA) {
	points := []image.Point{
		{cx, cy - longRadius},
		{cx + shortRadius/2, cy - shortRadius/2},
		{cx + longRadius, cy},
		{cx + shortRadius/2, cy + shortRadius/2},
		{cx, cy + longRadius},
		{cx - shortRadius/2, cy + shortRadius/2},
		{cx - longRadius, cy},
		{cx - shortRadius/2, cy - shortRadius/2},
	}
	fillPolygon(dst, points, value)
}

func fillPolygon(dst *image.RGBA, points []image.Point, value color.RGBA) {
	minY, maxY := points[0].Y, points[0].Y
	for _, point := range points[1:] {
		if point.Y < minY {
			minY = point.Y
		}
		if point.Y > maxY {
			maxY = point.Y
		}
	}
	for y := minY; y <= maxY; y++ {
		var intersections []int
		for i, a := range points {
			b := points[(i+1)%len(points)]
			if (a.Y <= y && b.Y > y) || (b.Y <= y && a.Y > y) {
				x := a.X + (y-a.Y)*(b.X-a.X)/(b.Y-a.Y)
				intersections = append(intersections, x)
			}
		}
		for i := 0; i+1 < len(intersections); i += 2 {
			left, right := intersections[i], intersections[i+1]
			if left > right {
				left, right = right, left
			}
			for x := left; x <= right; x++ {
				dst.SetRGBA(x, y, value)
			}
		}
	}
}

func writeICO(path string, pngData []byte) error {
	var ico bytes.Buffer
	_ = binary.Write(&ico, binary.LittleEndian, uint16(0))  // reserved
	_ = binary.Write(&ico, binary.LittleEndian, uint16(1))  // icon
	_ = binary.Write(&ico, binary.LittleEndian, uint16(1))  // one image
	ico.WriteByte(0)                                        // 256 px
	ico.WriteByte(0)                                        // 256 px
	ico.WriteByte(0)                                        // palette
	ico.WriteByte(0)                                        // reserved
	_ = binary.Write(&ico, binary.LittleEndian, uint16(1))  // planes
	_ = binary.Write(&ico, binary.LittleEndian, uint16(32)) // bpp
	_ = binary.Write(&ico, binary.LittleEndian, uint32(len(pngData)))
	_ = binary.Write(&ico, binary.LittleEndian, uint32(22))
	ico.Write(pngData)
	return os.WriteFile(path, ico.Bytes(), 0o644)
}
