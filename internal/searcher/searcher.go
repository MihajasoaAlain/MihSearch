package searcher

import (
	"github.com/miih/miih-search/internal/models"
	"github.com/miih/miih-search/internal/storage"
)

type Searcher struct {
	storage storage.Storage
}

func NewSearcher(storage storage.Storage) *Searcher {
	return &Searcher{
		storage: storage,
	}
}
func (s *Searcher) Search(
	word string) ([]models.SearchResult, error) {
	postings, err := s.storage.FindPostingByTerm(word)
	if err != nil {
		return nil, err
	}
	results := make([]models.SearchResult, 0)

	for _, posting := range postings {
		result := models.SearchResult{
			DocumentID: posting.DocumentID,
			Frequency:  posting.Frequency,
			Field:      posting.Field,
			Score:      float64(posting.Frequency),
		}
		results = append(results, result)
	}
	return results, nil
}
