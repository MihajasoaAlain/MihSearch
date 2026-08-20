package engine

import (
	"sort"

	"github.com/miih/miih-search/cmd/analyzer"
	"github.com/miih/miih-search/internal/models"
)

// Index writes a document and its inverted index entries. Title and content are
// analyzed separately so the searcher can tell where a term matched.
func (s *SearchEngine) Index(doc models.Document) error {
	if err := s.storage.SaveDocument(doc); err != nil {
		return err
	}

	documentID, err := s.storage.GetDocument(doc.ID)
	if err != nil {
		return err
	}

	// Indexing is idempotent: drop the previous postings so a term removed from
	// the document stops matching it.
	if err := s.storage.DeletePostingsByDocument(documentID); err != nil {
		return err
	}
	if err := s.storage.DeleteFieldLengthsByDocument(documentID); err != nil {
		return err
	}

	fields := []struct {
		name string
		text string
	}{
		{models.FieldTitle, doc.Title},
		{models.FieldContent, doc.Content},
	}

	for _, field := range fields {
		postings, length := analyzeField(field.text, field.name)

		// BM25 divides a term's frequency by the length of the field it was
		// found in, so the length has to be recorded while it is known.
		if err := s.storage.SaveFieldLength(documentID, field.name, length); err != nil {
			return err
		}

		for _, posting := range postings {
			if err := s.storage.SaveTerm(models.Term{Word: posting.word}); err != nil {
				return err
			}
			termID, err := s.storage.GetTermID(posting.word)
			if err != nil {
				return err
			}
			posting.posting.DocumentID = documentID
			if err := s.storage.SavePosting(posting.posting, termID, documentID); err != nil {
				return err
			}
		}
	}
	return nil
}

type termPosting struct {
	word    string
	posting models.Posting
}

// analyzeField collapses a field into one posting per distinct term, carrying
// the number of occurrences and every position where the term was found. The
// second return value is the field's length: how many terms it contributed to
// the index, which is what BM25 normalizes against.
func analyzeField(text string, field string) ([]termPosting, int) {
	tokens := analyzer.AnalyzeTokens(text)
	byWord := make(map[string]*models.Posting)
	for _, token := range tokens {
		posting, exists := byWord[token.Word]
		if !exists {
			byWord[token.Word] = &models.Posting{
				Frequency: 1,
				Positions: []int{token.Position},
				Field:     field,
			}
			continue
		}
		posting.Frequency++
		posting.Positions = append(posting.Positions, token.Position)
	}

	postings := make([]termPosting, 0, len(byWord))
	for word, posting := range byWord {
		postings = append(postings, termPosting{word: word, posting: *posting})
	}
	// Map iteration is random; writing terms in a stable order keeps indexing
	// runs reproducible.
	sort.Slice(postings, func(i, j int) bool {
		return postings[i].word < postings[j].word
	})
	return postings, len(tokens)
}
