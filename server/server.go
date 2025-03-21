package server

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"math"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
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
	logfile_name string
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

	data, err := io.ReadAll(file)
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

	server := &Server{name, port, image_folder, ImageProcessor{}, logger, images, &jsonMut, &loglineMut, loglines, "server/log.txt"}
	server.startLogWriter()
	return server
}

func (s *Server) startLogWriter() {

	go func() {
		//the new line is meant to be there
		s.log(fmt.Sprintf("\n\t%v", time.Now()))
		for {
			time.Sleep(10 * time.Second)

			s.loglineMut.Lock()

			if len(s.loglines) > 0 {
				err := s.writeLogLines()

				if err != nil {
					s.log(fmt.Sprintf("could not write log lines: %v", err))
				}
			}
			s.loglineMut.Unlock()

		}
	}()
}

func (s *Server) writeLogLines() error {

	file, err := os.OpenFile(s.logfile_name, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return fmt.Errorf("could not find or create file: %s %v", s.logfile_name, err)
	}
	defer file.Close()

	for _, logline := range s.loglines {
		if _, err := file.WriteString(time.TimeOnly + "\t" + logline); err != nil {
			return err
		}
	}
	s.loglines = []string{}

	return nil
}

func (s *Server) log(msg string) {
	s.logger.Printf("%s\n", msg)

	s.loglineMut.Lock()
	s.loglines = append(s.loglines, msg+"\n")
	s.loglineMut.Unlock()
}

func (s *Server) Handler(w http.ResponseWriter, r *http.Request) {
	go func() {
		msg := "Received request at /"
		s.log(msg)
		w.Write([]byte("Hello, World!"))
	}()
}

func (s *Server) handleProcess(w http.ResponseWriter, r *http.Request) {

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		s.log(fmt.Sprintf("could not parse form: %v", err))
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving the ile", http.StatusBadRequest)
		s.log(fmt.Sprintf("could not retrieve file: %v", err))
	}
	defer file.Close()

	action := r.FormValue("action")
	newName := r.FormValue("name")

	if action == "" {
		http.Error(w, "Missing action paramter", http.StatusBadRequest)
		s.log("missing action parameter")
	}

	tempFile, e := os.CreateTemp("", "upload-*")

	
	if e != nil {
		http.Error(w, "could not create temp file", http.StatusInternalServerError)
		s.log(fmt.Sprintf("could not create temp file: %v", err))
	}
	defer tempFile.Close()

	fileBytes, err := io.ReadAll(file)
	s.log(handler.Filename)
	if err != nil {
		http.Error(w, "could not read file", http.StatusInternalServerError)
		s.log(fmt.Sprintf("could not read file: %v", err))
	}

	tempFile.Write(fileBytes)
	tempFile.Seek(0, 0)

	s.log(fmt.Sprintf("received file: %s, action: %s", handler.Filename, action))
	img, format, err := image.Decode(tempFile)
	if err != nil {
		http.Error(w, "could not decode image", http.StatusInternalServerError)
		s.log(fmt.Sprintf("could not decode image: %v, format: %v", err, format))
	}
	fmt.Println("Howdy how are ya?",format)
	rgbaImg := s.imageToRGBA(img)

	s.log(fmt.Sprintf("received from url: %v", r.URL))

	var processed *image.RGBA
	switch action {
	case INVERT:
		processed = s.processor.GetInverted(*rgbaImg)
	case BLUR:
		processed = s.processor.GetBlurred(*rgbaImg)
	case GRAY:
		processed = s.processor.GetGray(*rgbaImg)
	case SOBEL:
		processed = s.processor.GetEdges(*rgbaImg)
	}

	var outfile_name string
	if newName != "" {
		outfile_name = newName + "." + format
	} else {
		outfile_name = action + "_" + newName + "." + format
	}

	res := s.saveImg(outfile_name, action, format, processed)

	var result ResponseData

	if res != nil {
		result = ResponseData{
			Message: fmt.Sprintf("Could not process and save file '%s' with action '%s'\n", outfile_name, action),
			Status:  "failure",
		}
		s.log(fmt.Sprintf("Couldn not save file '%s'", outfile_name))
	} else {

		result = ResponseData{
			Message: fmt.Sprintf("Processed file '%s' with action '%s'", outfile_name, action),
			Status:  "success",
		}
		s.log(fmt.Sprintf("Saved file '%s' to '%s'", handler.Filename, s.image_folder+"/"+outfile_name))
	}

	s.sendResponse(w, result, processed, format)
	s.log("sent response")
}

func (s *Server) sendResponse(w http.ResponseWriter, result ResponseData, processed *image.RGBA, format string) {
		writer := multipart.NewWriter(w)
	defer writer.Close()

	realBoundary := writer.Boundary()
	w.Header().Set("Content-Type", "multipart/mixed; boundary="+realBoundary)

	jsonPart, err := writer.CreatePart(textproto.MIMEHeader{"Content-Type": {"application/json"}})
	if err != nil {
		http.Error(w, "Error creating JSON part", http.StatusInternalServerError)
		s.log(fmt.Sprintf("could not create JSON part: %v", err))
	}
	json.NewEncoder(jsonPart).Encode(result)

	s.log(format)
	imgPart, err := writer.CreatePart(textproto.MIMEHeader{"Content-Type": {"image/" + format}})
	if err != nil {
		http.Error(w, "Error creating image part", http.StatusInternalServerError)
		s.log(fmt.Sprintf("could not create image part: %v", err))
	}

	switch format {
	case "png":
		png.Encode(imgPart, processed)
	case "jpeg":
		opts := jpeg.Options{Quality: 80}
		jpeg.Encode(imgPart, processed, &opts)
	default:
		http.Error(w, "Could not encode image", http.StatusInternalServerError)
		s.log(fmt.Sprintf("Could not encode image in response: %s", format))
	}

}

func (s *Server) imageToRGBA(img image.Image) *image.RGBA {

	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)

	return rgba
}

func (s *Server) saveImg(outfile_name, filter, suffix string, processed *image.RGBA) error {

	outfile, err := os.Create(s.image_folder + "/" + outfile_name)
	if err != nil {
		s.log(fmt.Sprintf("could not open file: %s, err:%v", outfile_name, err))
		return fmt.Errorf("could not create file %s", outfile_name)
	}
	defer outfile.Close()

	switch suffix {
	case "png":
		png.Encode(outfile, processed)
	case "jpeg":
		opts := jpeg.Options{Quality: 80}
		jpeg.Encode(outfile, processed, &opts)
	default:
		s.log(fmt.Sprintf("Could not encode. Suffix was not a supported type(%s)\n", suffix))
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
		s.log("error marshaling images")
		return fmt.Errorf("error marshaling images (%v)", err)

	}

	s.jsonMut.Lock()
	file, err := os.Create("server/imginfo.json")
	if err != nil {
		s.log("error opening imginfo.json")
		return fmt.Errorf("error opening imginfo.json")
	}
	defer file.Close()

	_, err = file.Write(jsonData)
	if err != nil {
		s.log("Error writing JSON to file: imginfo.json")
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
		s.log(fmt.Sprintf("could not delete image: %s", path))
		return fmt.Errorf("failed to delete file %s: %v ", path, err)
	}
	s.log(fmt.Sprintf("deleted image: %s from folder %s\n", path, s.image_folder))

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

	if err != nil {
		fmt.Println("Error opening file:", err)
		s.log(fmt.Sprintf("could not open file: %s, %v", path, err))
		return nil, err
	}
	defer file.Close()

	// Decode the image
	img, _, err := image.Decode(file)
	if err != nil {
		fmt.Println("Error decoding image:", err)
		return nil, err
	}

	return s.imageToRGBA(img), nil
}
