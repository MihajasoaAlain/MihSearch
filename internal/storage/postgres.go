package storage

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
)

type PostgresStorage struct {
	db *pgx.Conn
}

func NewPostgresStorage() (*PostgresStorage, error) {
	url := os.Getenv("DATABASE_URL")
	coon, err := pgx.Connect(context.Background(), url)
	if err != nil {
		return nil, err
	}
	fmt.Println("connected to postgres")
	return &PostgresStorage{
		db: coon,
	}, nil
}
