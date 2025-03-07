package img

import (
	"fmt"
	"golang.org/x/exp/constraints"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"os"
	"strings"
)

func max[T constraints.Ordered](a, b T) T {
	if a > b {
		return a
	}
	return b
}

func SaveImg(img *image.RGBA, name string) {

	outfile, err := os.Create(name)
	if err != nil {
		panic(err)
	}

	defer outfile.Close()

	split_name := strings.Split(name, ".")
	suffix := split_name[1]

	switch suffix {
	case "png":
		fmt.Println("save png")
		png.Encode(outfile, img)
	case "jpg":
		opts := &jpeg.Options{Quality: 80}
		fmt.Println("save jpeg")
		jpeg.Encode(outfile, img, opts)
	}

	fmt.Println("saved img to %s", name)

}

func InvertImg(img *image.RGBA) {
	bounds := img.Bounds()
	Max := bounds.Max
	width, height := Max.X, Max.Y

	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			c := img.At(x, y)
			r, g, b, a := c.RGBA()
			new_color := color.RGBA{R: uint8(255 - r), G: uint8(255 - g), B: uint8(255 - b), A: uint8(a)}
			img.Set(x, y, new_color)

		}
	}

}

func GetInverted(img image.RGBA) *image.RGBA {
	bounds := img.Bounds()
	inverted := image.NewRGBA(bounds)

	for i := bounds.Min.Y; i < bounds.Max.Y; i++ {
		for j := bounds.Min.X; j < bounds.Max.X; j++ {
			original_color := img.At(j, i)
			r, g, b, a := original_color.RGBA()

			inverted.Set(j, i, color.RGBA{
				R: uint8(255 - r>>8),
				G: uint8(255 - g>>8),
				B: uint8(255 - b>>8),
				A: uint8(a >> 8),
			})
		}
	}

	return inverted
}

func GetBlurred(img image.RGBA) *image.RGBA {

	bounds := img.Bounds()
	blurred := image.NewRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {

			r, g, b, a, n := uint32(0), uint32(0), uint32(0), uint32(0), uint32(0)
			for i := max(0, y-1); i < min(bounds.Max.Y, y+1); i++ {

				for j := max(0, x-1); j < min(bounds.Max.X, x+1); j++ {

					n += 1
					original_color := img.At(x, y)
					rr, gg, bb, aa := original_color.RGBA()
					r += rr
					g += gg
					b += bb
					a += aa

				}

			}

			blurred.Set(x, y, color.RGBA{
				R: uint8(r / n),
				G: uint8(g / n),
				B: uint8(b / n),
				A: uint8(a / n),
			})
		}

	}

	return blurred
}

func LoadImg(path string) (*image.RGBA, error) { // Open the image file
	file, err := os.Open(path) // Replace with your image file name
	image.RegisterFormat("png", "\x89PNG\r\n\x1a\n", png.Decode, png.DecodeConfig)
	image.RegisterFormat("jpeg", "\xff\xd8", jpeg.Decode, jpeg.DecodeConfig)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return nil, err
	}
	defer file.Close()

	// Decode the image
	img, format, err := image.Decode(file)
	if err != nil {
		fmt.Println("Error decoding image:", err)
		panic(err)
	}

	// Output image format
	fmt.Println("Image format:", format)

	// Get image dimensions
	bounds := img.Bounds()
	fmt.Printf("Image width: %d, height: %d\n", bounds.Dx(), bounds.Dy())

	new_RGBA := image.NewRGBA(bounds)
	draw.Draw(new_RGBA, new_RGBA.Bounds(), img, image.Point{0, 0}, draw.Src)

	return new_RGBA, nil
}
