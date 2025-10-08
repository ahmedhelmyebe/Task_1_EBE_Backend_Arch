package main

import (
	"log"
	"net/http"
)

func main() {
	fs := http.FileServer(http.Dir("."))
	http.Handle("/", fs)
	log.Println("UI on http://localhost:8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
