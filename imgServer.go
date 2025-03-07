package main

import (
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"net/http"
	"os"
	"strings"
)

func invertImage(img image.Image) *image.RGBA {
	bounds := img.Bounds()
	inverted := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			originalColor := img.At(x, y)
			r, g, b, a := originalColor.RGBA()
			inverted.Set(x, y, color.RGBA{
				R: uint8(255 - r>>8),
				G: uint8(255 - g>>8),
				B: uint8(255 - b>>8),
				A: uint8(a >> 8),
			})
		}
	}
	return inverted
}

func imageHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	fileName := q.Get("file")
	action := q.Get("action")

	if fileName == "" || action == "" {
		http.Error(w, "Missing file name or action.", http.StatusBadRequest)
		return
	}

	file, err := os.Open(fileName)
	if err != nil {
		http.Error(w, "File not found.", http.StatusNotFound)
		return
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		http.Error(w, "Error decoding image.", http.StatusInternalServerError)
		return
	}

	var processedImg image.Image
	if action == "invert" {
		processedImg = invertImage(img)
	} else {
		http.Error(w, "Invalid action.", http.StatusBadRequest)
		return
	}

	newFileName := strings.Split(fileName, ".")[0] + "Inverted.jpg"
	outFile, err := os.Create(newFileName)
	if err != nil {
		http.Error(w, "Error creating output file.", http.StatusInternalServerError)
		return
	}
	defer outFile.Close()

	err = jpeg.Encode(outFile, processedImg, nil)
	if err != nil {
		http.Error(w, "Error encoding image.", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "Image processed and saved as %s", newFileName)
}

func main() {
	http.HandleFunc("/process", imageHandler)
	fmt.Println("Server is listening on port 8080...")
	http.ListenAndServe(":8080", nil)
}
