package searcher

import (
	"github.com/miih/miih-search/cmd/analyzer"
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

//func (s *Searcher) Search(
//	query string) ([]models.SearchResult, error) {
//	postings, err := s.storage.FindPostingByTerm(word)
//	if err != nil {
//		return nil, err
//	}
//	results := make([]models.SearchResult, 0)
//
//	for _, posting := range postings {
//		result := models.SearchResult{
//			DocumentID: posting.DocumentID,
//			Frequency:  posting.Frequency,
//			Field:      posting.Field,
//			Score:      float64(posting.Frequency),
//		}
//		results = append(results, result)
//	}
//	return results, nil
//}

func (s *Searcher) Searcher(
	query string) ([]models.SearchResult, error) {
	words := analyzer.Analyzer(query)
	results := make(map[int]*models.SearchResult)
	for _, word := range words {
		postings, err := s.storage.FindPostingByTerm(word)
		if err != nil {
			return nil, err
		}
		for _, posting := range postings {
			doc, err := s.storage.GetDocumentByID(
				posting.DocumentID)
			if err != nil {
				return nil, err
			}
			result, exists := results[posting.DocumentID]
			if exists {
				result.Score += float64(posting.Frequency)

			} else {
				results[posting.DocumentID] = &models.SearchResult{DocumentID: doc.InternalID, ExternalID: doc.ID, Title: doc.Title, Content: doc.Content, Score: float64(posting.Frequency)}
			}
		}
	}
	final := make([]models.SearchResult, 0)
	for _, value := range results {
		final = append(final, *value)
	}
	return final, nil
}
