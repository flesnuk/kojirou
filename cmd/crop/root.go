package crop

import (
	"fmt"
	"image"
	"image/color"
	"math/bits"
)

const grayDarknessLimit = 128
const minDistinctBitsBetweenLines = 1 // minumum Hamming distance between consecutive line hashes to mark as border

func Auto(img image.Image) (image.Image, error) {
	bounds := BoundsHash(img)
	cropped, err := Crop(img, bounds)
	if err != nil {
		return nil, err
	}

	return cropped, nil
}

func Crop(img image.Image, bounds image.Rectangle) (image.Image, error) {
	type subImager interface {
		SubImage(r image.Rectangle) image.Image
	}

	if simg, ok := img.(subImager); !ok {
		return nil, fmt.Errorf("image does not support cropping")
	} else {
		return simg.SubImage(bounds), nil
	}
}

func Bounds(img image.Image) image.Rectangle {
	left := findBorder(img, image.Pt(1, 0))
	right := findBorder(img, image.Pt(-1, 0))
	top := findBorder(img, image.Pt(0, 1))
	bottom := findBorder(img, image.Pt(0, -1))

	return image.Rect(left.X, top.Y, right.X, bottom.Y)
}

func BoundsHash(img image.Image) image.Rectangle {
	left := findBorderUsingAvgHash(img, image.Pt(1, 0))
	right := findBorderUsingAvgHash(img, image.Pt(-1, 0))
	top := findBorderUsingAvgHash(img, image.Pt(0, 1))
	bottom := findBorderUsingAvgHash(img, image.Pt(0, -1))

	return image.Rect(left.X, top.Y, right.X, bottom.Y)
}

func findBorder(img image.Image, dir image.Point) image.Point {
	bounds := img.Bounds()
	scan := image.Pt(dir.Y, dir.X)
	dpt := pointInScanCorner(bounds, dir)

	for !scanLineForNonWhitespace(img, dpt, scan) {
		dpt = dpt.Add(dir)
		if !dpt.In(bounds) {
			dpt = pointInScanCorner(bounds, dir)
			break
		}
	}

	if dir.X < 0 || dir.Y < 0 {
		return dpt.Sub(dir)
	} else {
		return dpt
	}
}

func pointInScanCorner(rect image.Rectangle, dir image.Point) image.Point {
	if dir.X < 0 || dir.Y < 0 {
		return rect.Max.Sub(image.Pt(1, 1))
	} else {
		return rect.Min
	}
}

func scanLineForNonWhitespace(img image.Image, pt image.Point, scan image.Point) bool {
	for spt := pt; spt.In(img.Bounds()); spt = spt.Add(scan) {
		if gray, ok := color.GrayModel.Convert(img.At(spt.X, spt.Y)).(color.Gray); ok {
			if gray.Y <= grayDarknessLimit {
				return true
			}
		}
	}

	return false
}

func findBorderUsingAvgHash(img image.Image, dir image.Point) image.Point {
	bounds := img.Bounds()
	scan := image.Pt(dir.Y, dir.X)
	dpt := pointInScanCorner(bounds, dir)
	rgbimg, _ := img.(image.RGBA64Image)

	prev := lineAverageHash(rgbimg, dpt, scan)
	dpt = dpt.Add(dir)

	for {
		hash := lineAverageHash(rgbimg, dpt, scan)
		if bits.OnesCount64(hash^prev) >= minDistinctBitsBetweenLines {
			break
		}
		prev = hash
		dpt = dpt.Add(dir)
		if !dpt.In(bounds) {
			dpt = pointInScanCorner(bounds, dir)
			break
		}
	}

	if dir.X < 0 || dir.Y < 0 {
		return dpt.Sub(dir)
	} else {
		return dpt
	}
}

func lineAverageHash(img image.Image, pt image.Point, scan image.Point) uint64 {
	var hash uint64
	length := 0
	if scan.X != 0 {
		length = img.Bounds().Max.X
	} else {
		length = img.Bounds().Max.Y
	}
	windowSize := length / 64
	i := 0
	var partialSum uint32

	for spt := pt; spt.In(img.Bounds()); spt = spt.Add(scan) {
		if gray, ok := color.GrayModel.Convert(img.At(spt.X, spt.Y)).(color.Gray); ok {
			if i%windowSize == windowSize-1 {
				// check if average is "white" and set bit in hash, just before going into next block window.
				if partialSum > uint32(windowSize)*grayDarknessLimit {
					hash = setBit64(hash, i/windowSize)
				}
				partialSum = 0
			}
			partialSum += uint32(gray.Y)
			i++
		}
	}
	return hash
}

// Sets the bit at pos in the integer n.
func setBit64(n uint64, pos int) uint64 {
	n |= (1 << pos)
	return n
}
