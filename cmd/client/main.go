package main

import (
	"io"
	"log"
	"net/http"
)

func main() {
	log.Println("init")

	client := http.Client{}

	req, err := http.NewRequest("GET", "http://localhost:8080/", nil)
	if err != nil {
		log.Fatalf("failure on req: %v", err)
	}

	res, err := client.Do(req)
	if err != nil {
		log.Fatalf("failure on res: %v", err)
	}
	resBody := res.Body
	defer resBody.Close()
	resData, err := io.ReadAll(resBody)
	if (err != nil) {
		log.Fatalf("failure on res: %v", err)
	}

	log.Println("response:", string(resData))

	log.Println("quit")
}
