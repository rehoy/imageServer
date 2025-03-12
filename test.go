package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

type Image struct {
	Filename string `json:"filename"`
	Format   string `json:"format"`

	Original bool     `json:"original"`
	Filters  []string `json:"filters"`
}

func main() {
	fmt.Println("Hello World")

	filepath := "server/imginfo.json"

	file, err := os.Open(filepath)
	if err != nil {
		panic(err)
	}

	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatal("could not read file")
	}

	var images map[string]Image

	err = json.Unmarshal(data, &images)
	if err != nil {
		log.Fatal("Error unmarshaling JSON: %v", err)
	}

	fmt.Println(len(images))

	

}

