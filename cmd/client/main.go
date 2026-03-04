package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"repo.nikozdev.net/cryptowalletest/internal/model"
)

var baseURL = "http://localhost:8080"
var authToken string

func getUser(client *http.Client, id int) (*model.User, error) {
	url := fmt.Sprintf("%s/v1/users/%d", baseURL, id)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+authToken)
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("read failed: %w", err)
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d: %s", res.StatusCode, body)
	}
	var user model.User
	err = json.Unmarshal(body, &user)
	if err != nil {
		return nil, fmt.Errorf("decode failed: %w", err)
	}
	return &user, nil
}

func setUser(client *http.Client, id int, name string) error {
	url := fmt.Sprintf("%s/v1/users/%d", baseURL, id)
	payload, _ := json.Marshal(map[string]string{"v_name": name})
	req, err := http.NewRequest("PUT", url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+authToken)
	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("status %d: %s", res.StatusCode, body)
	}
	return nil
}

func main() {
	log.Println("init")

	authToken = os.Getenv("APP_AUTH_TOKEN")
	if authToken == "" {
		log.Fatal("APP_AUTH_TOKEN is not set")
	}

	client := &http.Client{}

	user, err := getUser(client, 1)
	if err != nil {
		log.Fatalf("get user: %v", err)
	}
	originalName := user.Name
	log.Printf("step 1: user name = %q", user.Name)

	err = setUser(client, 1, "user")
	if err != nil {
		log.Fatalf("set user: %v", err)
	}
	log.Println("step 2: name changed to \"user\"")

	user, err = getUser(client, 1)
	if err != nil {
		log.Fatalf("get user: %v", err)
	}
	log.Printf("step 3: user name = %q", user.Name)

	err = setUser(client, 1, originalName)
	if err != nil {
		log.Fatalf("set user: %v", err)
	}
	log.Printf("step 4: name restored to %q", originalName)

	user, err = getUser(client, 1)
	if err != nil {
		log.Fatalf("get user: %v", err)
	}
	log.Printf("step 5: user name = %q", user.Name)

	log.Println("quit")
}