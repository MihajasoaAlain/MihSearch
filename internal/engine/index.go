package engine

import (
	"github.com/miih/miih-search/internal/models"
	"github.com/miih/miih-search/internal/tokenizer"
)

func (s *SearchEngine) Index(doc models.Document) {
	words := tokenizer.Tokenize(doc.Content)
	for position, word := range words {
		s.addWord(word, doc.ID, position)
	}
}

func (s *SearchEngine) addWord(word string, documentID string, position int) {
	term, exists := s.index[word]
	if !exists {
		term = &models.Term{
			Word: word,
		}
		s.index[word] = term
	}
	var posting *models.Posting

	for i := range term.Postings {

		if term.Postings[i].DocumentID == documentID {

			posting = &term.Postings[i]

			break
		}
	}
	if posting == nil {
		term.Postings = append(term.Postings, models.Posting{
			DocumentID: documentID,
			Frequency:  0,
			Field:      "content",
		})
		posting = &term.Postings[len(term.Postings)-1]
	}
	posting.Frequency++
	posting.Positions = append(posting.Positions, position)
}
