// Command hash-password prints the Argon2id PHC string for a password read
// from stdin. It exists so deployment tooling can seed the admin account
// without duplicating the password format used by cmd/user-service.
package main

import (
	"fmt"
	"io"
	"os"

	authpassword "github.com/MAMUER/project/internal/auth/password"
)

func main() {
	plain, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "hash-password: read stdin:", err)
		os.Exit(1)
	}

	hash, err := authpassword.HashArgon2id(string(plain))
	if err != nil {
		fmt.Fprintln(os.Stderr, "hash-password:", err)
		os.Exit(1)
	}

	fmt.Print(hash)
}
