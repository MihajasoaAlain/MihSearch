package searcher

import (
	"sort"

	"github.com/miih/miih-search/cmd/analyzer"
	"github.com/miih/miih-search/internal/models"
	"github.com/miih/miih-search/internal/storage"
)

// fieldBoost weighs a match by where it happened: a term in the title says more
// about a document than the same term buried in the content.
var fieldBoost = map[string]float64{
	models.FieldTitle:   2.0,
	models.FieldContent: 1.0,
}

const defaultFieldBoost = 1.0

type Searcher struct {
	storage storage.Storage
}

func NewSearcher(storage storage.Storage) *Searcher {
	return &Searcher{
		storage: storage,
	}
}

// Search runs the query through the analysis pipeline and returns every document
// matching at least one term.
//
// Results are ranked by how many distinct query terms they matched first, then
// by score, so a document covering the whole query always outranks one that only
// matched a common word. The score itself sums frequency × field boost.
func (s *Searcher) Search(query string) ([]models.SearchResult, error) {
	terms := uniqueTerms(analyzer.Analyzer(query))
	if len(terms) == 0 {
		return []models.SearchResult{}, nil
	}

	postingsByTerm, err := s.storage.FindPostingsByTerms(terms)
	if err != nil {
		return nil, err
	}

	type accumulator struct {
		score        float64
		matchedTerms []string
	}
	accumulators := make(map[int]*accumulator)

	// Iterate over terms rather than over the map, so matched terms are reported
	// in query order.
	for _, term := range terms {
		for _, posting := range postingsByTerm[term] {
			boost, ok := fieldBoost[posting.Field]
			if !ok {
				boost = defaultFieldBoost
			}

			acc, exists := accumulators[posting.DocumentID]
			if !exists {
				acc = &accumulator{}
				accumulators[posting.DocumentID] = acc
			}
			acc.score += float64(posting.Frequency) * boost
			// The same term can match both fields of a document; count it once.
			if len(acc.matchedTerms) == 0 || acc.matchedTerms[len(acc.matchedTerms)-1] != term {
				acc.matchedTerms = append(acc.matchedTerms, term)
			}
		}
	}

	if len(accumulators) == 0 {
		return []models.SearchResult{}, nil
	}

	documentIDs := make([]int, 0, len(accumulators))
	for documentID := range accumulators {
		documentIDs = append(documentIDs, documentID)
	}
	documents, err := s.storage.GetDocumentsByIDs(documentIDs)
	if err != nil {
		return nil, err
	}

	results := make([]models.SearchResult, 0, len(accumulators))
	for _, documentID := range documentIDs {
		doc, exists := documents[documentID]
		if !exists {
			// The document was deleted between indexing and querying.
			continue
		}
		acc := accumulators[documentID]
		results = append(results, models.SearchResult{
			DocumentID:   documentID,
			ExternalID:   doc.ID,
			Type:         doc.Type,
			Title:        doc.Title,
			Content:      doc.Content,
			MatchedTerms: acc.matchedTerms,
			Score:        acc.score,
		})
	}

	sortResults(results)
	return results, nil
}

func sortResults(results []models.SearchResult) {
	sort.Slice(results, func(i, j int) bool {
		if len(results[i].MatchedTerms) != len(results[j].MatchedTerms) {
			return len(results[i].MatchedTerms) > len(results[j].MatchedTerms)
		}
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		// Last resort tie-break, so equally relevant documents keep a stable order.
		return results[i].ExternalID < results[j].ExternalID
	})
}

func uniqueTerms(words []string) []string {
	seen := make(map[string]struct{}, len(words))
	terms := make([]string, 0, len(words))
	for _, word := range words {
		if _, exists := seen[word]; exists {
			continue
		}
		seen[word] = struct{}{}
		terms = append(terms, word)
	}
	return terms
}
