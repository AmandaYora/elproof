// Package database opens the single shared MySQL connection pool for the process.
package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// Open converts the project's mysql:// DATABASE_URL (see .env.example) into the
// go-sql-driver/mysql DSN format and opens a pooled *sql.DB.
func Open(databaseURL string) (*sql.DB, error) {
	dsn, err := toMySQLDSN(databaseURL)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open mysql connection: %w", err)
	}

	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(30 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping mysql: %w", err)
	}

	return db, nil
}

// toMySQLDSN converts "mysql://user:pass@tcp(host:port)/dbname" (the .env.example
// convention) into the driver's "user:pass@tcp(host:port)/dbname?params" DSN shape.
func toMySQLDSN(databaseURL string) (string, error) {
	const prefix = "mysql://"
	if !strings.HasPrefix(databaseURL, prefix) {
		return "", fmt.Errorf("DATABASE_URL must start with %q, got %q", prefix, databaseURL)
	}
	dsn := strings.TrimPrefix(databaseURL, prefix)
	if strings.Contains(dsn, "?") {
		dsn += "&parseTime=true&loc=Local"
	} else {
		dsn += "?parseTime=true&loc=Local"
	}
	return dsn, nil
}
