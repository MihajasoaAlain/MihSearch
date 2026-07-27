package storage

import (
	"context"

	"github.com/miih/miih-search/internal/models"
)

func (p *PostgresStorage) SavePosting(
	posting models.Posting,
	termID int,
	documentID int) error {

	query := `
	INSERT INTO postings (term_id, document_id, frequency, positions, field)
	VALUES ($1, $2, $3, $4, $5)
	ON CONFLICT (term_id, document_id) 
	DO NOTHING;
	`
	_, err := p.db.Exec(
		context.Background(),
		query,
		termID,
		documentID,
		posting.Frequency,
		posting.Positions,
		posting.Field,
	)
	return err

}
