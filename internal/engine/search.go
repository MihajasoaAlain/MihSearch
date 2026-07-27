package engine

import "github.com/miih/miih-search/internal/models"

func (s *SearchEngine) Search(word string) []models.Posting {
	term, ok := s.index[word]
	if !ok {
		return nil
	}
	return term.Postings
}
