package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/spf13/cobra"

	"repo.nikozdev.net/cryptowalletest/internal/model"
)

var baseURL string
var authToken string

func doRequest(method, path string, body any) ([]byte, int, error) {
	var reqBody *bytes.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		reqBody = bytes.NewReader(data)
	} else {
		reqBody = bytes.NewReader(nil)
	}
	req, err := http.NewRequest(method, baseURL+path, reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+authToken)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, res.StatusCode, fmt.Errorf("read failed: %w", err)
	}
	return data, res.StatusCode, nil
}

func printJSON(data []byte) {
	var pretty bytes.Buffer
	if json.Indent(&pretty, data, "", "  ") == nil {
		fmt.Println(pretty.String())
	} else {
		fmt.Println(string(data))
	}
}

var rootCmd = &cobra.Command{
	Use:   "client",
	Short: "CryptoWalleTest CLI client",
}

var getUserCmd = &cobra.Command{
	Use:   "get-user [id]",
	Short: "Get user by ID",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		data, status, err := doRequest("GET", "/v1/users/"+args[0], nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if status != http.StatusOK {
			fmt.Fprintf(os.Stderr, "status %d: %s\n", status, data)
			os.Exit(1)
		}
		printJSON(data)
	},
}

var setUserCmd = &cobra.Command{
	Use:   "set-user [id] [name]",
	Short: "Update user name",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		payload := map[string]string{"name": args[1]}
		data, status, err := doRequest("PUT", "/v1/users/"+args[0], payload)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if status != http.StatusOK {
			fmt.Fprintf(os.Stderr, "status %d: %s\n", status, data)
			os.Exit(1)
		}
		fmt.Println("ok")
	},
}

var createWithdrawalCmd = &cobra.Command{
	Use:   "create-withdrawal",
	Short: "Create a new withdrawal request",
	Run: func(cmd *cobra.Command, args []string) {
		userID, _ := cmd.Flags().GetInt64("user-id")
		amount, _ := cmd.Flags().GetFloat64("amount")
		currency, _ := cmd.Flags().GetString("currency")
		destination, _ := cmd.Flags().GetString("destination")
		idempotencyKey, _ := cmd.Flags().GetString("key")

		payload := map[string]any{
			"user_id":         userID,
			"amount":          amount,
			"currency":        currency,
			"destination":     destination,
			"idempotency_key": idempotencyKey,
		}
		data, status, err := doRequest("POST", "/v1/withdrawals", payload)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if status != http.StatusCreated && status != http.StatusOK {
			fmt.Fprintf(os.Stderr, "status %d: %s\n", status, data)
			os.Exit(1)
		}
		var w model.Withdrawal
		json.Unmarshal(data, &w)
		if status == http.StatusOK {
			fmt.Println("idempotent replay:")
		}
		printJSON(data)
	},
}

var getWithdrawalCmd = &cobra.Command{
	Use:   "get-withdrawal [id]",
	Short: "Get withdrawal by ID",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		data, status, err := doRequest("GET", "/v1/withdrawals/"+args[0], nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if status != http.StatusOK {
			fmt.Fprintf(os.Stderr, "status %d: %s\n", status, data)
			os.Exit(1)
		}
		printJSON(data)
	},
}

var confirmWithdrawalCmd = &cobra.Command{
	Use:   "confirm-withdrawal [id]",
	Short: "Confirm a pending withdrawal",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		data, status, err := doRequest("POST", "/v1/withdrawals/"+args[0]+"/confirm", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if status != http.StatusOK {
			fmt.Fprintf(os.Stderr, "status %d: %s\n", status, data)
			os.Exit(1)
		}
		printJSON(data)
	},
}

func listRequest(cmd *cobra.Command, endpoint string) {
	limit, _ := cmd.Flags().GetInt("limit")
	offset, _ := cmd.Flags().GetInt("offset")
	path := fmt.Sprintf("%s?limit=%d&offset=%d", endpoint, limit, offset)
	data, status, err := doRequest("GET", path, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if status != http.StatusOK {
		fmt.Fprintf(os.Stderr, "status %d: %s\n", status, data)
		os.Exit(1)
	}
	printJSON(data)
}

var listUsersCmd = &cobra.Command{
	Use:   "list-users",
	Short: "List all users",
	Run: func(cmd *cobra.Command, args []string) {
		listRequest(cmd, "/v1/users")
	},
}

var listWithdrawalsCmd = &cobra.Command{
	Use:   "list-withdrawals",
	Short: "List all withdrawals",
	Run: func(cmd *cobra.Command, args []string) {
		listRequest(cmd, "/v1/withdrawals")
	},
}

var listLedgerCmd = &cobra.Command{
	Use:   "list-ledger",
	Short: "List ledger entries",
	Run: func(cmd *cobra.Command, args []string) {
		listRequest(cmd, "/v1/ledger")
	},
}

func addPaginationFlags(cmd *cobra.Command) {
	cmd.Flags().Int("limit", 20, "max number of results")
	cmd.Flags().Int("offset", 0, "number of results to skip")
}

func init() {
	rootCmd.PersistentFlags().StringVar(&baseURL, "url", "http://localhost:8080", "server base URL")
	rootCmd.PersistentFlags().StringVar(&authToken, "token", "", "auth token (overrides APP_AUTH_TOKEN)")

	createWithdrawalCmd.Flags().Int64("user-id", 1, "user ID")
	createWithdrawalCmd.Flags().Float64("amount", 0, "withdrawal amount")
	createWithdrawalCmd.Flags().String("currency", "USDT", "currency")
	createWithdrawalCmd.Flags().String("destination", "", "destination wallet address")
	createWithdrawalCmd.Flags().String("key", "", "idempotency key")
	createWithdrawalCmd.MarkFlagRequired("amount")
	createWithdrawalCmd.MarkFlagRequired("destination")
	createWithdrawalCmd.MarkFlagRequired("key")

	addPaginationFlags(listUsersCmd)
	addPaginationFlags(listWithdrawalsCmd)
	addPaginationFlags(listLedgerCmd)

	rootCmd.AddCommand(getUserCmd)
	rootCmd.AddCommand(setUserCmd)
	rootCmd.AddCommand(listUsersCmd)
	rootCmd.AddCommand(createWithdrawalCmd)
	rootCmd.AddCommand(getWithdrawalCmd)
	rootCmd.AddCommand(confirmWithdrawalCmd)
	rootCmd.AddCommand(listWithdrawalsCmd)
	rootCmd.AddCommand(listLedgerCmd)
}

func main() {
	cobra.OnInitialize(func() {
		if authToken == "" {
			authToken = os.Getenv("APP_AUTH_TOKEN")
		}
		if authToken == "" {
			fmt.Fprintln(os.Stderr, "auth token required: use --token or set APP_AUTH_TOKEN")
			os.Exit(1)
		}
	})

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
