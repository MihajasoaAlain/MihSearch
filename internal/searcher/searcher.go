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
// matching at least one term, ranked by BM25.
//
// Each field of a document is scored on its own and the scores are combined
// with the field weights, so a title match still counts double. Rarity does the
// rest: a document matching two common words can legitimately rank below one
// matching a single rare word.
func (s *Searcher) Search(query string) ([]models.SearchResult, error) {
	terms := uniqueTerms(analyzer.Analyzer(query))
	if len(terms) == 0 {
		return []models.SearchResult{}, nil
	}

	postingsByTerm, err := s.storage.FindPostingsByTerms(terms)
	if err != nil {
		return nil, err
	}
	if len(postingsByTerm) == 0 {
		return []models.SearchResult{}, nil
	}

	stats, err := s.storage.GetIndexStats()
	if err != nil {
		return nil, err
	}

	documentIDs := candidateDocuments(terms, postingsByTerm)
	fieldLengths, err := s.storage.GetFieldLengths(documentIDs)
	if err != nil {
		return nil, err
	}

	type accumulator struct {
		score        float64
		matchedTerms []string
	}
	accumulators := make(map[int]*accumulator, len(documentIDs))

	// Iterate over terms rather than over the map, so matched terms are reported
	// in query order.
	for _, term := range terms {
		postings := postingsByTerm[term]
		if len(postings) == 0 {
			continue
		}

		// The posting list is the term's whole presence in the index, so its
		// document frequency is already in hand: no extra query needed.
		weight := idf(stats.DocumentCount, documentFrequency(postings))

		for _, posting := range postings {
			boost, ok := fieldBoost[posting.Field]
			if !ok {
				boost = defaultFieldBoost
			}

			score := weight * termScore(
				posting.Frequency,
				fieldLengths[posting.DocumentID][posting.Field],
				stats.AverageFieldLength[posting.Field],
			) * boost

			acc, exists := accumulators[posting.DocumentID]
			if !exists {
				acc = &accumulator{}
				accumulators[posting.DocumentID] = acc
			}
			acc.score += score
			// The same term can match both fields of a document; count it once.
			if len(acc.matchedTerms) == 0 || acc.matchedTerms[len(acc.matchedTerms)-1] != term {
				acc.matchedTerms = append(acc.matchedTerms, term)
			}
		}
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

// candidateDocuments lists every document touched by the query, in a stable
// order so that equally scored results do not shuffle between runs.
func candidateDocuments(terms []string, postingsByTerm map[string][]models.Posting) []int {
	seen := make(map[int]struct{})
	documentIDs := make([]int, 0)
	for _, term := range terms {
		for _, posting := range postingsByTerm[term] {
			if _, exists := seen[posting.DocumentID]; exists {
				continue
			}
			seen[posting.DocumentID] = struct{}{}
			documentIDs = append(documentIDs, posting.DocumentID)
		}
	}
	sort.Ints(documentIDs)
	return documentIDs
}

// documentFrequency counts the documents a term appears in, not its postings:
// a term found in both the title and the content of one document still only
// covers a single document.
func documentFrequency(postings []models.Posting) int {
	documents := make(map[int]struct{}, len(postings))
	for _, posting := range postings {
		documents[posting.DocumentID] = struct{}{}
	}
	return len(documents)
}

func sortResults(results []models.SearchResult) {
	sort.Slice(results, func(i, j int) bool {
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
