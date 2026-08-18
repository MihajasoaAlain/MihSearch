package storage

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/miih/miih-search/internal/models"
)

// SaveDocument inserts the document, or refreshes it when it is already known.
// Re-indexing an existing external ID must pick up the new title and content.
func (p *PostgresStorage) SaveDocument(doc models.Document) error {
	query := `
	INSERT INTO documents (external_id, type, title, content)
	VALUES ($1, $2, $3, $4)
	ON CONFLICT (external_id)
	DO UPDATE SET
		type = EXCLUDED.type,
		title = EXCLUDED.title,
		content = EXCLUDED.content;
	`
	_, err := p.db.Exec(
		context.Background(),
		query,
		doc.ID,
		doc.Type,
		doc.Title,
		doc.Content,
	)
	return err
}

func (p *PostgresStorage) GetDocument(externalID string) (int, error) {

	var id int
	query := `
	SELECT id FROM documents WHERE external_id = $1;
	`
	err := p.db.QueryRow(
		context.Background(),
		query,
		externalID,
	).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	return id, err

}

func (p *PostgresStorage) GetDocumentByID(id int) (models.Document, error) {
	query := `
SELECT id, external_id, type, title, content FROM documents WHERE id = $1; `
	var doc models.Document
	err := p.db.QueryRow(
		context.Background(),
		query,
		id,
	).Scan(&doc.InternalID, &doc.ID, &doc.Type, &doc.Title, &doc.Content)
	if errors.Is(err, pgx.ErrNoRows) {
		return doc, ErrNotFound
	}
	return doc, err
}

// GetDocumentsByIDs loads a whole result page in one round trip, so ranking a
// query over N documents does not cost N queries.
func (p *PostgresStorage) GetDocumentsByIDs(ids []int) (map[int]models.Document, error) {
	documents := make(map[int]models.Document, len(ids))
	if len(ids) == 0 {
		return documents, nil
	}

	query := `
	SELECT id, external_id, type, title, content
	FROM documents
	WHERE id = ANY($1);
	`
	rows, err := p.db.Query(context.Background(), query, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var doc models.Document
		if err := rows.Scan(&doc.InternalID, &doc.ID, &doc.Type, &doc.Title, &doc.Content); err != nil {
			return nil, err
		}
		documents[doc.InternalID] = doc
	}
	return documents, rows.Err()
}
