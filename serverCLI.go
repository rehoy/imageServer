package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
)

func main() {
	// Command-line flags
	command := flag.String("command", "", "delete or process")
	fileName := flag.String("file", "", "Name of the image file to process")
	action := flag.String("action", "", "Action to perform on the image (e.g., invert)")
	name := flag.String("name", "", "name to save the processed as")
	flag.Parse()

	// Validate inputs
	if *fileName == "" || *action == "" {
		fmt.Println("Both file name and action are required.")
		flag.Usage()
		os.Exit(1)
	}

	// Construct the request URL
	baseURL := "http://localhost:8080/process"
	requestURL, err := url.Parse(baseURL)
	if err != nil {
		fmt.Println("Error parsing URL:", err)
		os.Exit(1)
	}

	// Set query parameters
	query := requestURL.Query()
	query.Set("file", *fileName)
	query.Set("action", *action)
	query.Set("name", *name)
	requestURL.RawQuery = query.Encode()

	// Make the HTTP request
	resp, err := http.Get(requestURL.String())
	if err != nil {
		fmt.Println("Error making HTTP request:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	// Process the response
	if resp.StatusCode == http.StatusOK {
		fmt.Println("Request successful:", resp.Status)
	} else {
		fmt.Printf("Request failed with status: %s\n", resp.Status)
	}
}
