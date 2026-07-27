package storage

import (
	"context"
)

func (p *PostgresStorage) CreateTables() error {

	query := `

	CREATE TABLE IF NOT EXISTS documents (
		id SERIAL PRIMARY KEY,
		external_id VARCHAR(255) UNIQUE,
		type VARCHAR(100),
		title TEXT,
		content TEXT
	);


	CREATE TABLE IF NOT EXISTS terms (
		id SERIAL PRIMARY KEY,
		word VARCHAR(255) UNIQUE
	);


	CREATE TABLE IF NOT EXISTS postings (
		id SERIAL PRIMARY KEY,
		term_id INT REFERENCES terms(id),
		document_id INT REFERENCES documents(id),
		frequency INT,
		positions INT[],
		field VARCHAR(50),
		UNIQUE(term_id, document_id)
	);

	`

	_, err := p.db.Exec(
		context.Background(),
		query,
	)

	return err
}
