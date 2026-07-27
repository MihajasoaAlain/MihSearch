package engine

import (
	"github.com/miih/miih-search/internal/models"
)

type SearchEngine struct {
	index map[string]*models.Term
}

func NewSearchEngine() *SearchEngine {
	return &SearchEngine{
		index: make(map[string]*models.Term),
	}
}
