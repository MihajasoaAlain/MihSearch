package storage

import (
	"context"

	"github.com/miih/miih-search/internal/models"
)

func (p *PostgresStorage) SaveDocument(doc models.Document) error {
	query := `
	INSERT INTO documents (external_id, type, title, content)
	VALUES ($1, $2, $3, $4)
	ON CONFLICT (external_id) 
	DO NOTHING;
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
	return doc, err
}
