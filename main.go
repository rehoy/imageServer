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
	http.HandleFunc("/process", s.ProcessHandler)
	http.HandleFunc("/delete", s.DeleteHandler)

	fmt.Printf("Listening on port %s\n", port)
	http.ListenAndServe(port, nil)

}
