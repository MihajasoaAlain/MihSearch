package engine

import (
	"github.com/miih/miih-search/internal/models"
	"github.com/miih/miih-search/internal/tokenizer"
)

type SearchEngine struct {
	index map[string][]string
}

func NewSearchEngine() *SearchEngine {
	return &SearchEngine{
		index: make(map[string][]string),
	}
}
func (s *SearchEngine) Index(doc models.Document) {
	words := tokenizer.Tokenize(doc.Content)
	for _, word := range words {
		s.index[word] = append(s.index[word], doc.ID)
	}
}
func (s *SearchEngine) Search(word string) []string {
	return s.index[word]
}
