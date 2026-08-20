package searcher

import (
	"strings"

	"github.com/miih/miih-search/cmd/analyzer"
	"github.com/miih/miih-search/internal/models"
)

// phraseTerm is one term of a quoted group, with the offset it holds relative
// to the first term of that group. The offset comes from the query's own token
// stream, so a stop word inside the quotes widens the gap the document has to
// reproduce — the same gap the indexer left behind when it dropped that word.
type phraseTerm struct {
	word   string
	offset int
}

// parsedQuery separates what the user quoted from what they did not. Terms
// holds every distinct term of the whole query, in query order, and is what the
// posting lookup runs on; phrases additionally constrain which documents may
// appear at all.
type parsedQuery struct {
	terms   []string
	phrases [][]phraseTerm
}

// parseQuery splits a raw query on double quotes and analyzes each part.
//
// Quotes have to be read before the pipeline runs, because punctuation removal
// erases them. Segments alternate: outside the quotes, then inside, then
// outside again. An unclosed quote leaves a trailing segment that never becomes
// a phrase — treating it as free text is friendlier than rejecting a query the
// user is still typing.
func parseQuery(raw string) parsedQuery {
	segments := strings.Split(raw, `"`)
	balanced := len(segments)%2 == 1

	parsed := parsedQuery{}
	seen := make(map[string]struct{})

	addTerm := func(word string) {
		if _, exists := seen[word]; exists {
			return
		}
		seen[word] = struct{}{}
		parsed.terms = append(parsed.terms, word)
	}

	for i, segment := range segments {
		isPhrase := i%2 == 1 && (balanced || i < len(segments)-1)
		if !isPhrase {
			for _, word := range analyzer.Analyzer(segment) {
				addTerm(word)
			}
			continue
		}

		tokens := analyzer.AnalyzeTokens(segment)
		if len(tokens) == 0 {
			// Nothing survived the pipeline — `"the of"` constrains nothing.
			continue
		}

		phrase := make([]phraseTerm, 0, len(tokens))
		for _, token := range tokens {
			addTerm(token.Word)
			phrase = append(phrase, phraseTerm{
				word:   token.Word,
				offset: token.Position - tokens[0].Position,
			})
		}
		parsed.phrases = append(parsed.phrases, phrase)
	}

	return parsed
}

// documentsMatchingPhrases returns the documents where every phrase of the
// query occurs, or nil when the query holds no phrase at all — the caller reads
// nil as "no restriction", which is not the same as an empty set.
//
// Quotes are a filter, not a hint: a document holding all the words in the
// wrong order is not what was asked for, so it is dropped rather than ranked
// lower. Ranking then proceeds normally over what is left.
func documentsMatchingPhrases(phrases [][]phraseTerm, postingsByTerm map[string][]models.Posting) map[int]struct{} {
	if len(phrases) == 0 {
		return nil
	}

	var allowed map[int]struct{}
	for _, phrase := range phrases {
		matching := documentsMatchingPhrase(phrase, postingsByTerm)
		if allowed == nil {
			allowed = matching
			continue
		}
		// Several quoted groups in one query all have to hold.
		for documentID := range allowed {
			if _, exists := matching[documentID]; !exists {
				delete(allowed, documentID)
			}
		}
	}
	return allowed
}

// documentsMatchingPhrase finds the documents where the phrase occurs inside a
// single field. A phrase straddling the title and the content is not a phrase:
// positions are recorded per field, and the two fields are unrelated texts.
func documentsMatchingPhrase(phrase []phraseTerm, postingsByTerm map[string][]models.Posting) map[int]struct{} {
	matching := make(map[int]struct{})
	if len(phrase) == 0 {
		return matching
	}

	// positions[i] maps a field of a document to the places the phrase's i-th
	// term was found there. A term missing from the index makes the whole
	// phrase impossible, so there is nothing left to look for.
	positions := make([]map[fieldKey]map[int]struct{}, len(phrase))
	for i, term := range phrase {
		postings := postingsByTerm[term.word]
		if len(postings) == 0 {
			return matching
		}
		positions[i] = positionsByField(postings)
	}

	// The first term anchors the search: every place it appears is a candidate
	// start, and the others only have to confirm the offset the query asked for.
	for key, starts := range positions[0] {
		for start := range starts {
			if phraseOccursAt(positions, key, start, phrase) {
				matching[key.documentID] = struct{}{}
				break
			}
		}
	}
	return matching
}

func phraseOccursAt(positions []map[fieldKey]map[int]struct{}, key fieldKey, start int, phrase []phraseTerm) bool {
	for i := 1; i < len(phrase); i++ {
		found, exists := positions[i][key]
		if !exists {
			return false
		}
		if _, exists := found[start+phrase[i].offset]; !exists {
			return false
		}
	}
	return true
}

// fieldKey identifies the one text a phrase can span: a given field of a given
// document.
type fieldKey struct {
	documentID int
	field      string
}

func positionsByField(postings []models.Posting) map[fieldKey]map[int]struct{} {
	byField := make(map[fieldKey]map[int]struct{}, len(postings))
	for _, posting := range postings {
		key := fieldKey{documentID: posting.DocumentID, field: posting.Field}
		found, exists := byField[key]
		if !exists {
			found = make(map[int]struct{}, len(posting.Positions))
			byField[key] = found
		}
		for _, position := range posting.Positions {
			found[position] = struct{}{}
		}
	}
	return byField
}
