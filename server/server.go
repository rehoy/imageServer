package server

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	INVERT = "invert"
	SOBEL  = "sobel"
	GRAY   = "gray"
	BLUR   = "blur"
)

type ResponseData struct {
	Message string `json:"message"`
	Status  string `json:"status"`
}

type Server struct {
	name         string
	port         string
	image_folder string
	processor    ImageProcessor
	logger       *log.Logger
	images       map[string]Image
	jsonMut      *sync.Mutex
	loglineMut   *sync.Mutex
	loglines     []string
}

type ImageProcessor struct {
}

type Image struct {
	Filename string `json:"filename"`
	Format   string `json:"format"`

	Original bool     `json:"original"`
	Filters  []string `json:"filters"`
}

func readImages(filepath string) map[string]Image {

	file, err := os.Open(filepath)
	if err != nil {
		log.Fatalf("Could not open file %s: err: %v\n", filepath, err)
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatalf("could not read data, %v\n", err)
	}

	var images map[string]Image

	err = json.Unmarshal(data, &images)
	if err != nil {
		log.Fatalf("Could not unmarshal JSON err: %v\n", err)
	}
	return images
}

func NewServer(name, port, image_folder string) *Server {
	logger := log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)

	images := readImages("server/imginfo.json")

	var jsonMut sync.Mutex
	var loglineMut sync.Mutex
	var loglines []string

	server := &Server{name, port, image_folder, ImageProcessor{}, logger, images, &jsonMut, &loglineMut, loglines}
	server.startLogWriter()
	return server
}

func (s *Server) startLogWriter() {

	go func() {
		for {
			time.Sleep(10 * time.Second)

			fmt.Println("Writing Log")

		}
	}()
}

func (s *Server) log(msg string) {
	s.loglines = append(s.loglines, msg)
	s.logger.Printf(msg)
}

func (s *Server) Handler(w http.ResponseWriter, r *http.Request) {
	go func() {
		msg := "Received request at /\n"
		s.log(msg)
		w.Write([]byte("Hello, World!"))
	}()
}

func (s *Server) handleProcess(w http.ResponseWriter, r *http.Request) {
	image_folder := "images"
	queryParams := r.URL.Query()
	filename := queryParams.Get("file")
	action := queryParams.Get("action")
	new_name := queryParams.Get("name")

	//queryString := s.queryAsString(queryParams)

	s.logger.Println("received request at /process with params:", queryParams)

	if filename == "" || action == "" {
		s.logger.Printf("missing filename(%s) or or action(%s)\n", filename, action)
		http.Error(w, "missing filename or action in parameters", http.StatusBadRequest)
	}

	img, ok := s.LoadImg(image_folder + "/" + filename)
	if ok != nil {
		s.logger.Printf("could not load image %s\n", filename)
		http.Error(w, "file does not exist on server", http.StatusNotFound)
	}

	var filter string
	var processed *image.RGBA
	switch action {
	case INVERT:
		filter = INVERT
		processed = s.processor.GetInverted(*img)
	case BLUR:
		filter = BLUR
		processed = s.processor.GetBlurred(*img)
	case GRAY:
		filter = GRAY
		processed = s.processor.GetGray(*img)
	case SOBEL:
		filter = SOBEL
		processed = s.processor.GetEdges(*img)
	}

	split_filename := strings.Split(filename, ".")
	suffix := split_filename[1]

	var outfile_name string
	if new_name == "" {
		outfile_name = action + filename
	} else {
		outfile_name = new_name + "." + suffix
	}

	res := s.saveImg(outfile_name, filter, suffix, processed)

	var result ResponseData

	if res != nil {
		result = ResponseData{
			Message: fmt.Sprintf("Could not process and save file '%s' with action '%s'\n", outfile_name, action),
			Status:  "failure",
		}
		s.logger.Printf("Couldn not save file '%s'\n", outfile_name)
	} else {

		result = ResponseData{
			Message: fmt.Sprintf("Processed file '%s' with action '%s'", outfile_name, action),
			Status:  "success",
		}
		s.logger.Printf("Saved file '%s' to '%s'", filename, s.image_folder+"/"+outfile_name)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)

}

func (s *Server) saveImg(outfile_name, filter, suffix string, processed *image.RGBA) error {

	outfile, err := os.Create(s.image_folder + "/" + outfile_name)
	if err != nil {
		s.logger.Printf("could not open file: %s, err:%v\n", outfile_name, err)
		return fmt.Errorf("could not create file %s", outfile_name)
	}

	defer outfile.Close()

	switch suffix {
	case "png":
		png.Encode(outfile, processed)
	case "jpg":
		opts := jpeg.Options{Quality: 80}
		jpeg.Encode(outfile, processed, &opts)
	default:
		s.logger.Printf("Could not encode. Suffix was not a supported type(%s)\n", suffix)
		return fmt.Errorf("not of supported type: %s", suffix)
	}

	var image Image

	image, ok := s.images[outfile_name]

	if !ok {

		image = Image{
			Format:   suffix,
			Original: false,
			Filters:  []string{filter},
			Filename: outfile_name,
		}
	} else {
		image.Filters = append(image.Filters, filter)
	}

	s.images[outfile_name] = image

	go s.writeJSON()

	return nil
}

func (s *Server) writeJSON() error {

	images := s.images

	jsonData, err := json.MarshalIndent(images, "", "  ")
	if err != nil {
		s.logger.Printf("error marshaling images")
		return fmt.Errorf("error marshaling images (%v)", err)

	}

	s.jsonMut.Lock()
	file, err := os.Create("server/imginfo.json")
	if err != nil {
		s.logger.Printf("error opening imginfo.json")
		return fmt.Errorf("error opening imginfo.json")
	}
	defer file.Close()

	_, err = file.Write(jsonData)
	if err != nil {
		s.logger.Printf("error wrting JSON to file: imginfo.json")
		return fmt.Errorf("error writing JSON to file: %w", err)
	}
	s.jsonMut.Unlock()

	return nil

}

func (s *Server) ProcessHandler(w http.ResponseWriter, r *http.Request) {
	s.handleProcess(w, r)

}

func (s *Server) deleteImage(path string) error {

	filepath := s.image_folder + "/" + path

	err := os.Remove(filepath)
	if err != nil {
		s.logger.Printf("could not delete image: %s\n", path)
		return fmt.Errorf("failed to delete file %s: %v ", path, err)
	}
	s.logger.Printf("deleted image: %s from folder %s\n", path, s.image_folder)

	return nil

}

func (s *Server) DeleteHandler(w http.ResponseWriter, r *http.Request) {

	queryParams := r.URL.Query()
	file := queryParams.Get("file")

	if file == "" {

		http.Error(w, "file parameter not", http.StatusBadRequest)

	}

	err := s.deleteImage(file)

	if err != nil {
		http.Error(w, "could not delete file", http.StatusBadRequest)
	}

	fmt.Fprintf(w, "deleted %s", file)
}

func (p *ImageProcessor) InvertImg(img *image.RGBA) {
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

func (p *ImageProcessor) GetInverted(img image.RGBA) *image.RGBA {
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

func (p *ImageProcessor) GetBlurred(img image.RGBA, args ...int) *image.RGBA {

	var blurStrength int

	if len(args) == 0 {
		blurStrength = 7
	} else {
		blurStrength = args[0]
	}

	bounds := img.Bounds()
	blurred := image.NewRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {

			r, g, b, a, n := uint32(0), uint32(0), uint32(0), uint32(0), uint32(0)
			for i := max(0, y-blurStrength); i < min(bounds.Max.Y, y+blurStrength); i++ {

				for j := max(0, x-blurStrength); j < min(bounds.Max.X, x+blurStrength); j++ {

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

func (p *ImageProcessor) GetGray(img image.RGBA) *image.RGBA {
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

func (p *ImageProcessor) GetEdges(img image.RGBA) *image.RGBA {

	gray_img := p.GetGray(img)

	bounds := img.Bounds()
	gray := image.NewGray(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {

			gray_color := gray_img.At(x, y)
			gray.Set(x, y, gray_color)

		}
	}

	horizontal_kernel := [3][3]int{{-1, 0, 1}, {-2, 0, 2}, {-1, 0, 1}}
	vertical_kernel := [3][3]int{{-1, -2, -1}, {0, 0, 0}, {1, 2, 1}}

	sobel := image.NewRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			valX := 0 // Sum for horizontal kernel
			valY := 0
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
						horizontal_kernelValue := horizontal_kernel[ky+1][kx+1]
						vertical_kernelValue := vertical_kernel[ky+1][kx+1]

						neighborIntensity := int(gray.GrayAt(neighborX, neighborY).Y)
						valX += horizontal_kernelValue * neighborIntensity
						valY += vertical_kernelValue * neighborIntensity
					}
				}
			}
			magnitude := math.Sqrt(float64(valX*valX) + float64(valY*valY))

			// Convert to grayscale and clamp
			abs_value := uint8(clamp(int(magnitude), 0, 255))
			sobel.Set(x, y, color.RGBA{abs_value, abs_value, abs_value, 255})
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

func (s *Server) LoadImg(path string) (*image.RGBA, error) { // Open the image file
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
		return nil, err
	}

	// Output image format

	// Get image dimensions
	bounds := img.Bounds()

	new_RGBA := image.NewRGBA(bounds)
	draw.Draw(new_RGBA, new_RGBA.Bounds(), img, image.Point{0, 0}, draw.Src)

	return new_RGBA, nil
}
