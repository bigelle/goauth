package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Database interface {
	CreateUser(ctx context.Context, name, email, pass string) error
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByName(ctx context.Context, username string) (*User, error)
	Close() error
}

const (
	SQLITE3_driver = "sqlite3"
)

func NewDatabase(driver string, dsn string) (Database, error) {
	switch driver {
	case SQLITE3_driver:
		return newSqlite3Db(dsn)
	}
	return nil, fmt.Errorf("unsupported driver: %s", driver)
}

type User struct {
	UUID         string    `db:"id"`
	Name         string    `db:"name"`
	Email        string    `db:"email"`
	PasswordHash string    `db:"password"`
	CreatedAt    time.Time `db:"created_at"`
}

var (
	ErrNotFound = errors.New("user not found")
)
