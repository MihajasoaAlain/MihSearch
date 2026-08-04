package storage

import (
	"context"

	"github.com/miih/miih-search/internal/models"
)

func (p *PostgresStorage) FindPostingByTerm(word string) ([]models.Posting, error) {
	query := `
SELECT 
p.document_id,
p.frequency,
p.positions,
p.field
FROM postings p

JOIN terms t ON p.term_id = t.id
WHERE t.word = $1;`

	rows, err := p.db.Query(
		context.Background(), query, word)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var postings []models.Posting
	for rows.Next() {
		var posting models.Posting
		err := rows.Scan(&posting.DocumentID, &posting.Frequency, &posting.Positions, &posting.Field)
		if err != nil {
			return nil, err
		}
		postings = append(postings, posting)
	}
	return postings, nil
}
