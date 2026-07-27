package engine

import (
	"github.com/miih/miih-search/internal/models"
	"github.com/miih/miih-search/internal/tokenizer"
)

func (s *SearchEngine) Index(doc models.Document) error {
	err := s.storage.SaveDocument(doc)
	if err != nil {
		return err
	}

	documentId, err := s.storage.GetDocument(doc.ID)
	if err != nil {
		return err
	}

	words := tokenizer.Tokenize(doc.Content)
	for position, word := range words {
		err = s.storage.SaveTerm(models.Term{Word: word})
		if err != nil {
			return err
		}
		termID, err := s.storage.GetTermID(word)
		if err != nil {
			return err
		}
		posting := models.Posting{
			DocumentID: doc.ID,
			Frequency:  1,
			Positions:  []int{position},
			Field:      "content",
		}

		err = s.storage.SavePosting(posting, termID, documentId)
		if err != nil {
			return err
		}

	}
	return nil
}
