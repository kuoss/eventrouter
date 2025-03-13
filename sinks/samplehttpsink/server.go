package main

import (
	"bufio"
	"io"
	"log"
	"net/http"
	"strings"
)

func handler(w http.ResponseWriter, r *http.Request) {
	log.Printf("request method=%s from=%s", r.Method, r.RemoteAddr)
	if r.Body == nil {
		return
	}
	defer r.Body.Close()

	reader := bufio.NewReader(r.Body)
	for {
		msg, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		} else if err != nil {
			log.Fatalf("Error reading message: %v", err)
		}
		msg = strings.TrimSpace(msg) // Trim the newline from the end
		if len(msg) == 0 {
			continue // Skip empty lines
		}
		log.Print(msg)
	}
}

func main() {
	log.Println("starting httpsink server")
	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
