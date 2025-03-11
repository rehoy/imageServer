package main

import (
	"fmt"
	"net/http"
	"path/filepath"
)

func main() {

	staticDir := filepath.Join(".", "webinterface")

	fs := http.FileServer(http.Dir(staticDir))

	http.Handle("/", fs)

	port := ":8081"
	fmt.Printf("Serving static files at localhost:%s\n", port)

	if err := http.ListenAndServe(port, nil); err != nil {
		fmt.Printf("Error starting server:%s\n", err)
	}

}
