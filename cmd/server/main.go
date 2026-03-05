package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"

	"repo.nikozdev.net/cryptowalletest/internal/database"
	"repo.nikozdev.net/cryptowalletest/internal/model"
)

func getPagination(r *http.Request) (int, int) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

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
		Name string `json:"name"`
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

func createWithdrawalHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		UserID         int64   `json:"user_id"`
		Amount         float64 `json:"amount"`
		Currency       string  `json:"currency"`
		Destination    string  `json:"destination"`
		IdempotencyKey string  `json:"idempotency_key"`
	}
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if input.IdempotencyKey == "" {
		http.Error(w, "idempotency key required", http.StatusBadRequest)
		return
	}
	if input.Amount <= 0 {
		http.Error(w, "amount must be positive", http.StatusBadRequest)
		return
	}
	if input.Currency == "" {
		input.Currency = "USDT"
	}
	if input.Currency != "USDT" {
		http.Error(w, "unsupported currency", http.StatusBadRequest)
		return
	}

	var existing model.Withdrawal
	err = db.QueryRow(
		`SELECT v_id, v_user_id, v_amount, v_currency, v_destination, v_status, v_idempotency_key, v_created_at
		FROM t_withdrawal WHERE v_idempotency_key = $1`,
		input.IdempotencyKey,
	).Scan(
		&existing.ID, &existing.UserID, &existing.Amount,
		&existing.Currency, &existing.Destination, &existing.Status,
		&existing.IdempotencyKey, &existing.CreatedAt,
	)
	if err == nil {
		if existing.UserID == input.UserID &&
			existing.Amount == input.Amount &&
			existing.Currency == input.Currency &&
			existing.Destination == input.Destination {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(existing)
			return
		}
		http.Error(w, "idempotency key conflict", http.StatusUnprocessableEntity)
		return
	}
	if err != sql.ErrNoRows {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var balance float64
	err = tx.QueryRow(
		`SELECT v_balance FROM t_user WHERE v_id = $1 FOR UPDATE`,
		input.UserID,
	).Scan(&balance)
	if err == sql.ErrNoRows {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if balance < input.Amount {
		http.Error(w, "insufficient balance", http.StatusConflict)
		return
	}

	newBalance := balance - input.Amount
	_, err = tx.Exec(
		`UPDATE t_user SET v_balance = $1 WHERE v_id = $2`,
		newBalance, input.UserID,
	)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	var withdrawal model.Withdrawal
	err = tx.QueryRow(
		`INSERT INTO t_withdrawal (v_user_id, v_amount, v_currency, v_destination, v_idempotency_key)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING v_id, v_user_id, v_amount, v_currency, v_destination, v_status, v_idempotency_key, v_created_at`,
		input.UserID, input.Amount, input.Currency, input.Destination, input.IdempotencyKey,
	).Scan(
		&withdrawal.ID, &withdrawal.UserID, &withdrawal.Amount,
		&withdrawal.Currency, &withdrawal.Destination, &withdrawal.Status,
		&withdrawal.IdempotencyKey, &withdrawal.CreatedAt,
	)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec(
		`INSERT INTO t_ledger_entry (v_user_id, v_withdrawal_id, v_type, v_amount, v_balance_after)
		VALUES ($1, $2, 'withdrawal', $3, $4)`,
		input.UserID, withdrawal.ID, input.Amount, newBalance,
	)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	err = tx.Commit()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(withdrawal)
}

func getWithdrawalHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var withdrawal model.Withdrawal
	err := db.QueryRow(
		`SELECT v_id, v_user_id, v_amount, v_currency, v_destination, v_status, v_idempotency_key, v_created_at
		FROM t_withdrawal WHERE v_id = $1`,
		id,
	).Scan(
		&withdrawal.ID, &withdrawal.UserID, &withdrawal.Amount,
		&withdrawal.Currency, &withdrawal.Destination, &withdrawal.Status,
		&withdrawal.IdempotencyKey, &withdrawal.CreatedAt,
	)
	if err == sql.ErrNoRows {
		http.Error(w, "withdrawal not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(withdrawal)
}

func confirmWithdrawalHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	tx, err := db.Begin()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var withdrawal model.Withdrawal
	err = tx.QueryRow(
		`SELECT v_id, v_user_id, v_amount, v_currency, v_destination, v_status, v_idempotency_key, v_created_at
		FROM t_withdrawal WHERE v_id = $1 FOR UPDATE`,
		id,
	).Scan(
		&withdrawal.ID, &withdrawal.UserID, &withdrawal.Amount,
		&withdrawal.Currency, &withdrawal.Destination, &withdrawal.Status,
		&withdrawal.IdempotencyKey, &withdrawal.CreatedAt,
	)
	if err == sql.ErrNoRows {
		http.Error(w, "withdrawal not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if withdrawal.Status != "pending" {
		http.Error(w, "withdrawal not pending", http.StatusConflict)
		return
	}

	_, err = tx.Exec(
		`UPDATE t_withdrawal SET v_status = 'confirmed' WHERE v_id = $1`,
		id,
	)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	err = tx.Commit()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	withdrawal.Status = "confirmed"
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(withdrawal)
}

func listUsersHandler(w http.ResponseWriter, r *http.Request) {
	limit, offset := getPagination(r)
	rows, err := db.Query(
		`SELECT v_id, v_name, v_balance, v_created_at FROM t_user ORDER BY v_id LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	users := []model.User{}
	for rows.Next() {
		var u model.User
		rows.Scan(&u.ID, &u.Name, &u.Balance, &u.CreatedAt)
		users = append(users, u)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func listWithdrawalsHandler(w http.ResponseWriter, r *http.Request) {
	limit, offset := getPagination(r)
	rows, err := db.Query(
		`SELECT v_id, v_user_id, v_amount, v_currency, v_destination, v_status, v_idempotency_key, v_created_at
		FROM t_withdrawal ORDER BY v_id LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	withdrawals := []model.Withdrawal{}
	for rows.Next() {
		var wd model.Withdrawal
		rows.Scan(&wd.ID, &wd.UserID, &wd.Amount, &wd.Currency, &wd.Destination, &wd.Status, &wd.IdempotencyKey, &wd.CreatedAt)
		withdrawals = append(withdrawals, wd)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(withdrawals)
}

func listLedgerHandler(w http.ResponseWriter, r *http.Request) {
	limit, offset := getPagination(r)
	rows, err := db.Query(
		`SELECT v_id, v_user_id, v_withdrawal_id, v_type, v_amount, v_balance_after, v_created_at
		FROM t_ledger_entry ORDER BY v_id LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	entries := []model.LedgerEntry{}
	for rows.Next() {
		var e model.LedgerEntry
		rows.Scan(&e.ID, &e.UserID, &e.WithdrawalID, &e.Type, &e.Amount, &e.BalanceAfter, &e.CreatedAt)
		entries = append(entries, e)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
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
	mux.HandleFunc("GET /v1/users", listUsersHandler)
	mux.HandleFunc("GET /v1/users/{id}", getUserHandler)
	mux.HandleFunc("PUT /v1/users/{id}", setUserHandler)
	mux.HandleFunc("GET /v1/withdrawals", listWithdrawalsHandler)
	mux.HandleFunc("POST /v1/withdrawals", createWithdrawalHandler)
	mux.HandleFunc("GET /v1/withdrawals/{id}", getWithdrawalHandler)
	mux.HandleFunc("POST /v1/withdrawals/{id}/confirm", confirmWithdrawalHandler)
	mux.HandleFunc("GET /v1/ledger", listLedgerHandler)

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}
	addr := "0.0.0.0:" + port
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, checkAuth(mux)))
}