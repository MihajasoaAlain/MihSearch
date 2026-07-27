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
