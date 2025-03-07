package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"net/http"
	"os"
	"strings"
)

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
			for i := max(0, y-10); i < min(bounds.Max.Y, y+10); i++ {

				for j := max(0, x-10); j < min(bounds.Max.X, x+10); j++ {

					n += 1
					original_color := img.At(j, i)
					rr, gg, bb, aa := original_color.RGBA()
					r += rr
					g += gg
					b += bb
					a += aa

				}

			}

			blurred.Set(x, y, color.RGBA{
				R: uint8((r / n) >> 8),
				G: uint8((g / n) >> 8),
				B: uint8((b / n) >> 8),
				A: uint8((a / n) >> 8),
			})
		}

	}

	return blurred
}

func GetGray(img image.RGBA) *image.RGBA {
	bounds := img.Bounds()
	gray := image.NewGray(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {

			original_color := img.At(x, y)
			gray.Set(x, y, original_color)

		}
	}

	gray_img := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {

			gray_color := gray.At(x, y)
			gray_img.Set(x, y, gray_color)

		}
	}
	return gray_img
}

func GetEdges(img image.RGBA) *image.RGBA {

	gray_img := GetGray(img)

	bounds := img.Bounds()
	gray := image.NewGray(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {

			gray_color := gray_img.At(x, y)
			gray.Set(x, y, gray_color)

		}
	}

	horizontal_kernel := [3][3]int{{-1, 0, 1}, {-2, 0, 2}, {-1, 0, 1}}
	//vertical_kernel := [3][3]int{{-1, -2 - 1}, {0, 0, 0}, {1, 2, 1}}

	sobel := image.NewRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			valX := 0 // Sum for horizontal kernel
			// Iterate over the 3x3 kernel
			for ky := -1; ky <= 1; ky++ {
				for kx := -1; kx <= 1; kx++ {
					// Calculate the position of the neighbor pixel
					neighborX := x + kx
					neighborY := y + ky

					// Check for image boundaries to avoid out-of-bounds access
					if neighborX >= bounds.Min.X && neighborX < bounds.Max.X &&
						neighborY >= bounds.Min.Y && neighborY < bounds.Max.Y {
						// Apply the horizontal kernel
						kernelValue := horizontal_kernel[ky+1][kx+1]
						neighborIntensity := int(gray.GrayAt(neighborX, neighborY).Y)
						valX += kernelValue * neighborIntensity
					}
				}
			}
			// Convert the result to a grayscale value and then RGBA
			g := uint8(clamp(valX, 0, 255)) // A helper function to clamp values
			sobel.Set(x, y, color.RGBA{g, g, g, 255})
		}
	}

	return sobel

}
func clamp(value, min, max int) int {
	if value < min {
		return min
	} else if value > max {
		return max
	}
	return value
}

func LoadImg(path string) (*image.RGBA, error) { // Open the image file
	file, err := os.Open(path) // Replace with your image file name
	image.RegisterFormat("", "\x89PNG\r\n\x1a\n", png.Decode, png.DecodeConfig)
	image.RegisterFormat("jpeg", "\xff\xd8", jpeg.Decode, jpeg.DecodeConfig)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return nil, err
	}
	defer file.Close()

	// Decode the image
	img, _, err := image.Decode(file)
	if err != nil {
		fmt.Println("Error decoding image:", err)
		panic(err)
	}

	// Output image format

	// Get image dimensions
	bounds := img.Bounds()

	new_RGBA := image.NewRGBA(bounds)
	draw.Draw(new_RGBA, new_RGBA.Bounds(), img, image.Point{0, 0}, draw.Src)

	return new_RGBA, nil
}

func handler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello, World!"))
}

func processHandler(w http.ResponseWriter, r *http.Request) {

	image_folder := "images"
	queryParams := r.URL.Query()
	filename := queryParams.Get("file")
	action := queryParams.Get("action")
	new_name := queryParams.Get("name")

	fmt.Printf("filename: %s, action: %s", filename, action)

	if filename == "" || action == "" {
		http.Error(w, "missing filename or action in parameters", http.StatusBadRequest)
	}

	img, ok := LoadImg(image_folder + "/" + filename)
	if ok != nil {
		http.Error(w, "file does not exist on server", http.StatusNotFound)
	}

	var processed *image.RGBA
	switch action {
	case "invert":
		processed = GetInverted(*img)
	case "blur":
		processed = GetBlurred(*img)
	case "gray":
		processed = GetGray(*img)
	case "sobel":
		processed = GetEdges(*img)
	}

	split_filename := strings.Split(filename, ".")
	suffix := split_filename[1]

	var outfile_name string
	if new_name == "" {
		outfile_name = action + filename
	} else {
		outfile_name = new_name + "." + suffix
	}

	outfile, ok := os.Create(image_folder + "/" + outfile_name)
	if ok != nil {
		http.Error(w, "could not create file", http.StatusConflict)
	}
	defer outfile.Close()

	switch suffix {
	case "png":
		png.Encode(outfile, processed)
	case "jpg":
		opts := jpeg.Options{Quality: 80}
		jpeg.Encode(outfile, processed, &opts)
	}

	fmt.Fprintf(w, "did action %s to file: %s and saved it as %s", action, filename, outfile_name)
}

func main() {
	port_number := flag.String("port", "", "port number")

	flag.Parse()

	if *port_number == "" {
		fmt.Println("no port number provided")
		flag.Usage()
		os.Exit(1)
	}
	port := ":" + *port_number

	http.HandleFunc("/", handler)
	http.HandleFunc("/process", processHandler)

	fmt.Printf("Listening on port %s\n", port)
	http.ListenAndServe(port, nil)

}
