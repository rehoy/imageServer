package main

import (
	"flag"
	"fmt"
	"github.uio.no/oysteere/myserver/server"
	"net/http"
	"os"
)

func main() {
	port_number := flag.String("port", "", "port number")

	flag.Parse()

	if *port_number == "" {
		fmt.Println("no port number provided")
		flag.Usage()
		os.Exit(1)
	}
	port := ":" + *port_number

	s := server.NewServer("server", port, "images")

	http.HandleFunc("/", s.Handler)
	http.HandleFunc("/process", corsMiddleware(s.ProcessHandler))
	http.HandleFunc("/delete", s.DeleteHandler)

	fmt.Printf("Listening on port %s\n", port)
	http.ListenAndServe(port, nil)

}
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Adjust to allow requests from 'http://localhost:8081'
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:8081")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// Handle preflight OPTIONS requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next(w, r)
	}
}
