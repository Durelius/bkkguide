package main

import (
	"bufio"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"

	"bkkguide/internal/audit"
	"bkkguide/internal/auth"
)

// createAdmin interactively adds an admin. It's how the first admin is made,
// and a way back in if everyone is locked out. Logged as admin "system".
func createAdmin(db *sql.DB) error {
	in := bufio.NewReader(os.Stdin)
	ask := func(prompt string) string {
		fmt.Print(prompt)
		line, _ := in.ReadString('\n')
		return strings.TrimSpace(line)
	}
	askSecret := func(prompt string) (string, error) {
		fmt.Print(prompt)
		if !term.IsTerminal(int(os.Stdin.Fd())) {
			line, err := in.ReadString('\n')
			return strings.TrimSpace(line), err
		}
		b, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		return string(b), err
	}

	id, err := auth.NormalizeStudentID(ask("Student ID: "))
	if err != nil {
		return err
	}
	name := ask("Full name: ")
	if name == "" {
		return errors.New("full name is required")
	}
	pw, err := askSecret(fmt.Sprintf("Password (min %d characters): ", auth.MinPasswordLen))
	if err != nil {
		return err
	}
	again, err := askSecret("Repeat password: ")
	if err != nil {
		return err
	}
	if pw != again {
		return errors.New("passwords don't match")
	}
	hash, err := auth.HashPassword(pw)
	if err != nil {
		return err
	}

	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT INTO admins (student_id, full_name, password_hash) VALUES (?, ?, ?)`, id, name, hash); err != nil {
		return fmt.Errorf("create admin: %w", err)
	}
	if err := audit.Log(ctx, tx, "system", "create", "admin", id, map[string]string{"fullName": name, "via": "command line"}); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	fmt.Printf("Created admin %s (%s).\n", id, name)
	return nil
}
