package engine

import (
	"github.com/miih/miih-search/internal/storage"
)

type SearchEngine struct {
	storage storage.Storage
}

func NewSearchEngine(db storage.Storage) *SearchEngine {
	return &SearchEngine{
		storage: db,
	}
}
