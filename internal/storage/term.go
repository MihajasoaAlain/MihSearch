package storage

import (
	"context"

	"github.com/miih/miih-search/internal/models"
)

func (p *PostgresStorage) SaveTerm(term models.Term) error {
	query := `
	INSERT INTO terms (word)
	VALUES ($1)
	ON CONFLICT (word) 
	DO NOTHING;
	`
	_, err := p.db.Exec(
		context.Background(),
		query,
		term.Word)

	return err
}

func (p *PostgresStorage) GetTermID(word string) (int, error) {
	var id int
	query := `
	SELECT id FROM terms WHERE word = $1;
	`
	err := p.db.QueryRow(
		context.Background(),
		query,
		word).Scan(&id)
	return id, err
}
