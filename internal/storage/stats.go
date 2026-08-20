package storage

import (
	"context"

	"github.com/miih/miih-search/internal/models"
)

// SaveFieldLength records how many terms a document's field contributed to the
// index. Re-indexing recomputes it, so a conflict overwrites the old value.
func (p *PostgresStorage) SaveFieldLength(documentID int, field string, length int) error {
	query := `
	INSERT INTO document_fields (document_id, field, length)
	VALUES ($1, $2, $3)
	ON CONFLICT (document_id, field)
	DO UPDATE SET length = EXCLUDED.length;
	`
	_, err := p.db.Exec(context.Background(), query, documentID, field, length)
	return err
}

func (p *PostgresStorage) DeleteFieldLengthsByDocument(documentID int) error {
	_, err := p.db.Exec(
		context.Background(),
		`DELETE FROM document_fields WHERE document_id = $1;`,
		documentID,
	)
	return err
}

// GetFieldLengths returns the indexed length of every field of the given
// documents, as documentID -> field -> length.
func (p *PostgresStorage) GetFieldLengths(documentIDs []int) (map[int]map[string]int, error) {
	lengths := make(map[int]map[string]int, len(documentIDs))
	if len(documentIDs) == 0 {
		return lengths, nil
	}

	query := `
	SELECT document_id, field, length
	FROM document_fields
	WHERE document_id = ANY($1);
	`
	rows, err := p.db.Query(context.Background(), query, documentIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var documentID, length int
		var field string
		if err := rows.Scan(&documentID, &field, &length); err != nil {
			return nil, err
		}
		if lengths[documentID] == nil {
			lengths[documentID] = make(map[string]int)
		}
		lengths[documentID][field] = length
	}
	return lengths, rows.Err()
}

// GetIndexStats reports the corpus size and the average length of each field.
func (p *PostgresStorage) GetIndexStats() (models.IndexStats, error) {
	stats := models.IndexStats{AverageFieldLength: make(map[string]float64)}

	err := p.db.QueryRow(
		context.Background(),
		`SELECT COUNT(*) FROM documents;`,
	).Scan(&stats.DocumentCount)
	if err != nil {
		return stats, err
	}

	rows, err := p.db.Query(
		context.Background(),
		`SELECT field, AVG(length)::float8 FROM document_fields GROUP BY field;`,
	)
	if err != nil {
		return stats, err
	}
	defer rows.Close()

	for rows.Next() {
		var field string
		var average float64
		if err := rows.Scan(&field, &average); err != nil {
			return stats, err
		}
		stats.AverageFieldLength[field] = average
	}
	return stats, rows.Err()
}
