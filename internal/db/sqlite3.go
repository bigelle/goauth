package db

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Sqlite3Database struct {
	db *sqlx.DB
}

const sqlite3Schema = `
CREATE TABLE IF NOT EXISTS users (
    id BLOB PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
	email TEXT NOT NULL COLLATE NOCASE UNIQUE,
	password TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
`

func newSqlite3Db(dsn string) (*Sqlite3Database, error) {
	if dir := filepath.Dir(dsn); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
	}

	db, err := sqlx.Open(SQLITE3_driver, dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	if _, err := db.Exec(sqlite3Schema); err != nil {
		return nil, err
	}

	return &Sqlite3Database{db: db}, nil
}

const sqlite3CreateUserRequest = `
INSERT INTO users (id, name, email, password)
VALUES (?, ?, ?, ?);
`

func (db *Sqlite3Database) CreateUser(ctx context.Context, name, email, pass string) error {
	uuid, err := uuid.NewV7()
	if err != nil {
		return err
	}

	if _, err = db.db.ExecContext(ctx, sqlite3CreateUserRequest,
		uuid, name, email, pass); err != nil {
		// FIXME: check for conflicts
		return err
	}

	return nil
}

const sqlite3GetUserByEmailRequest = `
SELECT * FROM users WHERE email = (?);
`

func (db *Sqlite3Database) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	err := db.db.GetContext(ctx, &user, sqlite3GetUserByEmailRequest, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &user, nil
}

func (db *Sqlite3Database) GetUserByName(ctx context.Context, username string) (*User, error) {
	return nil, errors.ErrUnsupported
}

func (db *Sqlite3Database) Close() error {
	return db.db.Close()
}
