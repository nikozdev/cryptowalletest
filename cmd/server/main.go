package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"repo.nikozdev.net/cryptowalletest/internal/database"
)

var db *sql.DB

type User struct {
	ID        int64     `json:"v_id"`
	Name      string    `json:"v_name"`
	Balance   float64   `json:"v_balance"`
	CreatedAt time.Time `json:"v_created_at"`
}

func getUserHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var user User
	err := db.QueryRow(
		`SELECT v_id, v_name, v_balance, v_created_at FROM t_user WHERE v_id = $1`,
		id,
	).Scan(&user.ID, &user.Name, &user.Balance, &user.CreatedAt)
	if err == sql.ErrNoRows {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func setUserHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var input struct {
		Name string `json:"v_name"`
	}
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	result, err := db.Exec(
		`UPDATE t_user SET v_name = $1 WHERE v_id = $2`,
		input.Name, id,
	)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func main() {
	log.Println("init")

	var err error
	db, err = database.GetDatabase()
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()
	log.Println("database connected")

	err = database.RunMigrations(db, "migrations")
	if err != nil {
		log.Fatalf("migrations: %v", err)
	}
	log.Println("migrations done")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/users/{id}", getUserHandler)
	mux.HandleFunc("PUT /v1/users/{id}", setUserHandler)

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}
	addr := "0.0.0.0:" + port
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}