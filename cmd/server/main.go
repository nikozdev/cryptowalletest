package main

import (
	"log"
	"net/http"
)

func getHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("request", r.URL)
	if (r.Method == "GET") {
		w.Write([]byte("hello world"))
	}
}

func main() {
	log.Println("init")

	mux := http.NewServeMux()
	mux.HandleFunc("/", getHandler)

	http.ListenAndServe("0.0.0.0:8080", mux);

	log.Println("quit")
}
