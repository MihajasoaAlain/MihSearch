package storage

import (
	"context"

	"github.com/miih/miih-search/internal/models"
)

// SavePosting writes the posting for one (term, document, field) triple. The
// frequency and positions are recomputed on every indexing pass, so a conflict
// overwrites the previous values instead of being ignored.
func (p *PostgresStorage) SavePosting(
	posting models.Posting,
	termID int,
	documentID int) error {

	query := `
	INSERT INTO postings (term_id, document_id, frequency, positions, field)
	VALUES ($1, $2, $3, $4, $5)
	ON CONFLICT (term_id, document_id, field)
	DO UPDATE SET
		frequency = EXCLUDED.frequency,
		positions = EXCLUDED.positions;
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

// DeletePostingsByDocument clears a document's postings before it is re-indexed,
// so terms that disappeared from the new version stop matching it.
func (p *PostgresStorage) DeletePostingsByDocument(documentID int) error {
	_, err := p.db.Exec(
		context.Background(),
		`DELETE FROM postings WHERE document_id = $1;`,
		documentID,
	)
	return err
}
