package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"repo.nikozdev.net/cryptowalletest/internal/database"
	"repo.nikozdev.net/cryptowalletest/internal/model"
)

var db *sql.DB
var authToken string

func checkAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if header == "" || header != "Bearer "+authToken {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func getUserHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var user model.User
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

	migrationsDir := os.Getenv("MIGRATIONS_DIR")
	if migrationsDir == "" {
		migrationsDir = "migrations"
	}
	err = database.RunMigrations(db, migrationsDir)
	if err != nil {
		log.Fatalf("migrations: %v", err)
	}
	log.Println("migrations done")

	authToken = os.Getenv("APP_AUTH_TOKEN")
	if authToken == "" {
		log.Fatal("APP_AUTH_TOKEN is not set")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/users/{id}", getUserHandler)
	mux.HandleFunc("PUT /v1/users/{id}", setUserHandler)

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}
	addr := "0.0.0.0:" + port
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, checkAuth(mux)))
}