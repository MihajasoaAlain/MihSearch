package engine

import (
	"github.com/miih/miih-search/internal/models"
	"github.com/miih/miih-search/internal/storage"
)

type SearchEngine struct {
	index   map[string]models.Term
	storage *storage.PostgresStorage
}

func NewSearchEngine(db *storage.PostgresStorage) *SearchEngine {
	return &SearchEngine{
		index:   make(map[string]models.Term),
		storage: db,
	}
}
