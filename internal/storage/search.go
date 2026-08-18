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
	return postings, rows.Err()
}

// FindPostingsByTerms resolves every term of a multi-term query in a single
// round trip and groups the postings by the term they came from.
func (p *PostgresStorage) FindPostingsByTerms(words []string) (map[string][]models.Posting, error) {
	grouped := make(map[string][]models.Posting, len(words))
	if len(words) == 0 {
		return grouped, nil
	}

	query := `
SELECT
t.word,
p.document_id,
p.frequency,
p.positions,
p.field
FROM postings p
JOIN terms t ON p.term_id = t.id
WHERE t.word = ANY($1)
ORDER BY p.document_id, p.field;`

	rows, err := p.db.Query(context.Background(), query, words)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var word string
		var posting models.Posting
		if err := rows.Scan(&word, &posting.DocumentID, &posting.Frequency, &posting.Positions, &posting.Field); err != nil {
			return nil, err
		}
		grouped[word] = append(grouped[word], posting)
	}
	return grouped, rows.Err()
}
