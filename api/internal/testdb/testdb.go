// Package testdb provides isolated, migrated PostgreSQL schemas for integration tests.
//
// Tests are skipped unless MK_TEST_DATABASE_URL points to a PostgreSQL database
// (URL form, e.g. postgres://user:pass@localhost:5432/db?sslmode=disable).
package testdb

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/samuelfabel/megumi-kura/api/internal/database"
)

const EnvURL = "MK_TEST_DATABASE_URL"

// Open creates a fresh schema, applies all migrations to it and returns a
// connection whose search_path points at that schema. The schema is dropped
// when the test finishes.
func Open(t *testing.T) *sql.DB {
	t.Helper()
	base := os.Getenv(EnvURL)
	if base == "" {
		t.Skipf("%s not set; skipping integration test", EnvURL)
	}

	admin, err := database.Open(base)
	if err != nil {
		t.Fatalf("open admin connection: %v", err)
	}

	schema := "t_" + randomHex(t, 8)
	if _, err := admin.Exec(`CREATE SCHEMA ` + schema); err != nil {
		_ = admin.Close()
		t.Fatalf("create schema: %v", err)
	}

	u, err := url.Parse(base)
	if err != nil {
		t.Fatalf("parse %s: %v", EnvURL, err)
	}
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()

	db, err := database.Open(u.String())
	if err != nil {
		t.Fatalf("open test connection: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
		_, _ = admin.Exec(`DROP SCHEMA ` + schema + ` CASCADE`)
		_ = admin.Close()
	})

	if err := database.Migrate(context.Background(), db, migrationsDir(t)); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

// Food inserts an active food and returns its id.
func Food(t *testing.T, db *sql.DB, code, name, unit string) int64 {
	t.Helper()
	var id int64
	if err := db.QueryRow(`
		INSERT INTO foods(code, name, unit) VALUES ($1, $2, $3) RETURNING id`,
		code, name, unit).Scan(&id); err != nil {
		t.Fatalf("insert food %s: %v", code, err)
	}
	return id
}

func migrationsDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate testdb source file")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "migrations", "postgres")
}

func randomHex(t *testing.T, n int) string {
	t.Helper()
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("random: %v", err)
	}
	return hex.EncodeToString(b)
}
