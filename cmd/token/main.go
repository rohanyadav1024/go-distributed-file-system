package main

import (
	"fmt"
	"os"
	"time"

	"github.com/rohanyadav1024/dfs/internal/auth"
)

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
