package storage

import (
	"context"
)

// migrations run in order on every startup and must all be idempotent.
var migrations = []string{
	`CREATE TABLE IF NOT EXISTS documents (
		id SERIAL PRIMARY KEY,
		external_id VARCHAR(255) UNIQUE,
		type VARCHAR(100),
		title TEXT,
		content TEXT
	);`,

	`CREATE TABLE IF NOT EXISTS terms (
		id SERIAL PRIMARY KEY,
		word VARCHAR(255) UNIQUE
	);`,

	`CREATE TABLE IF NOT EXISTS postings (
		id SERIAL PRIMARY KEY,
		term_id INT REFERENCES terms(id),
		document_id INT REFERENCES documents(id),
		frequency INT,
		positions INT[],
		field VARCHAR(50) NOT NULL DEFAULT 'content'
	);`,

	// Field-aware indexing: a term now has one posting per field, so the old
	// (term, document) uniqueness has to give way to (term, document, field).
	`ALTER TABLE postings ALTER COLUMN field SET DEFAULT 'content';`,
	`UPDATE postings SET field = 'content' WHERE field IS NULL;`,
	`ALTER TABLE postings ALTER COLUMN field SET NOT NULL;`,
	`ALTER TABLE postings DROP CONSTRAINT IF EXISTS postings_term_id_document_id_key;`,
	`CREATE UNIQUE INDEX IF NOT EXISTS postings_term_document_field_key
		ON postings (term_id, document_id, field);`,

	`CREATE INDEX IF NOT EXISTS postings_term_id_idx ON postings (term_id);`,
	`CREATE INDEX IF NOT EXISTS postings_document_id_idx ON postings (document_id);`,
}

func (p *PostgresStorage) CreateTables() error {
	for _, migration := range migrations {
		if _, err := p.db.Exec(context.Background(), migration); err != nil {
			return err
		}
	}
	return nil
}
