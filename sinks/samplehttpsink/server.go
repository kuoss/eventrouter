package main

import (
	"io"
	"log"
	"net/http"

	"github.com/kuoss/eventrouter/sinks/rfc5424"
)

func handler(w http.ResponseWriter, r *http.Request) {
	log.Printf("request method=%s from=%s", r.Method, r.RemoteAddr)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusInternalServerError)
		return
	}
	defer func() {
		if err := r.Body.Close(); err != nil {
			log.Printf("r.Body.Close() error: %v", err)
		}
	}()

	m, err := rfc5424.NewFromBytes(body)
	if err != nil {
		log.Fatalf("Parsing rfc5424 message failed: %+v", err)
	}
	log.Printf("%s", m.Message)
}

func main() {
	log.Println("starting httpsink server")
	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
