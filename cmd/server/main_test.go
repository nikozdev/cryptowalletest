package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"repo.nikozdev.net/cryptowalletest/internal/database"
	"repo.nikozdev.net/cryptowalletest/internal/model"
)

var testServer *httptest.Server

func TestMain(m *testing.M) {
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:17",
		postgres.WithDatabase("d_test"),
		postgres.WithUsername("u_test"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2),
		),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start postgres: %v\n", err)
		os.Exit(1)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get connection string: %v\n", err)
		os.Exit(1)
	}

	db, err = sql.Open("pgx", connStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open database: %v\n", err)
		os.Exit(1)
	}

	_, filename, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(filename), "..", "..")
	migrationsDir := filepath.Join(projectRoot, "migrations")
	err = database.RunMigrations(db, migrationsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to run migrations: %v\n", err)
		os.Exit(1)
	}

	authToken = "test-token"

	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/users/{id}", getUserHandler)
	mux.HandleFunc("PUT /v1/users/{id}", setUserHandler)
	mux.HandleFunc("POST /v1/withdrawals", createWithdrawalHandler)
	mux.HandleFunc("GET /v1/withdrawals/{id}", getWithdrawalHandler)
	mux.HandleFunc("POST /v1/withdrawals/{id}/confirm", confirmWithdrawalHandler)
	testServer = httptest.NewServer(checkAuth(mux))

	code := m.Run()

	testServer.Close()
	pgContainer.Terminate(ctx)
	os.Exit(code)
}

func doRequest(method, path string, body interface{}) *http.Response {
	var reqBody *bytes.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		reqBody = bytes.NewReader(data)
	} else {
		reqBody = bytes.NewReader(nil)
	}
	req, _ := http.NewRequest(method, testServer.URL+path, reqBody)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	resp, _ := http.DefaultClient.Do(req)
	return resp
}

func TestCreateWithdrawalSuccess(t *testing.T) {
	db.Exec(`UPDATE t_user SET v_balance = 1000.0 WHERE v_id = 1`)

	payload := map[string]interface{}{
		"user_id":         1,
		"amount":          100.0,
		"currency":        "USDT",
		"destination":     "0xabc123",
		"idempotency_key": "success-test-key",
	}
	resp := doRequest("POST", "/v1/withdrawals", payload)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	var withdrawal model.Withdrawal
	json.NewDecoder(resp.Body).Decode(&withdrawal)
	if withdrawal.Status != "pending" {
		t.Fatalf("expected status pending, got %s", withdrawal.Status)
	}
	if withdrawal.Amount != 100.0 {
		t.Fatalf("expected amount 100, got %f", withdrawal.Amount)
	}
}

func TestCreateWithdrawalInsufficientBalance(t *testing.T) {
	db.Exec(`UPDATE t_user SET v_balance = 50.0 WHERE v_id = 1`)

	payload := map[string]interface{}{
		"user_id":         1,
		"amount":          999999.0,
		"currency":        "USDT",
		"destination":     "0xdef456",
		"idempotency_key": "insufficient-test-key",
	}
	resp := doRequest("POST", "/v1/withdrawals", payload)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}
}

func TestCreateWithdrawalIdempotency(t *testing.T) {
	db.Exec(`UPDATE t_user SET v_balance = 1000.0 WHERE v_id = 1`)

	payload := map[string]interface{}{
		"user_id":         1,
		"amount":          10.0,
		"currency":        "USDT",
		"destination":     "0xidemp",
		"idempotency_key": "idempotency-test-key",
	}

	resp1 := doRequest("POST", "/v1/withdrawals", payload)
	defer resp1.Body.Close()
	if resp1.StatusCode != http.StatusCreated {
		t.Fatalf("first call expected 201, got %d", resp1.StatusCode)
	}
	var w1 model.Withdrawal
	json.NewDecoder(resp1.Body).Decode(&w1)

	resp2 := doRequest("POST", "/v1/withdrawals", payload)
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("second call expected 200, got %d", resp2.StatusCode)
	}
	var w2 model.Withdrawal
	json.NewDecoder(resp2.Body).Decode(&w2)

	if w1.ID != w2.ID {
		t.Fatalf("expected same ID, got %d and %d", w1.ID, w2.ID)
	}
}

func TestCreateWithdrawalConcurrent(t *testing.T) {
	db.Exec(`UPDATE t_user SET v_balance = 100.0 WHERE v_id = 1`)

	concurrency := 10
	amount := 50.0
	var wg sync.WaitGroup
	results := make([]int, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			payload := map[string]interface{}{
				"user_id":         1,
				"amount":          amount,
				"currency":        "USDT",
				"destination":     "0xconcurrent",
				"idempotency_key": fmt.Sprintf("concurrent-key-%d", idx),
			}
			resp := doRequest("POST", "/v1/withdrawals", payload)
			results[idx] = resp.StatusCode
			resp.Body.Close()
		}(i)
	}
	wg.Wait()

	successCount := 0
	conflictCount := 0
	for _, code := range results {
		switch code {
		case http.StatusCreated:
			successCount++
		case http.StatusConflict:
			conflictCount++
		}
	}

	if successCount > 2 {
		t.Fatalf("expected at most 2 successes (100/50), got %d", successCount)
	}
	if successCount+conflictCount != concurrency {
		t.Fatalf("expected all responses to be 201 or 409, got mixed")
	}

	var balance float64
	db.QueryRow(`SELECT v_balance FROM t_user WHERE v_id = 1`).Scan(&balance)
	expectedBalance := 100.0 - (float64(successCount) * amount)
	if balance != expectedBalance {
		t.Fatalf("expected balance %f, got %f", expectedBalance, balance)
	}
}
