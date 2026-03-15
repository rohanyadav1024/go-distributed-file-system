// Package main prints a short-lived JWT for local testing.
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/rohanyadav1024/dfs/internal/auth"
)

// main generates and prints a client token using the configured secret.
func main() {
	secret := os.Getenv("DFS_JWT_SECRET")
	if secret == "" {
		panic("DFS_JWT_SECRET is required and cannot be empty")
	}

	token, err := auth.GenerateToken(secret, "client", time.Hour)
	if err != nil {
		panic(err)
	}

	fmt.Println(token)
}
