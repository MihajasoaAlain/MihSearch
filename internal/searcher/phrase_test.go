package searcher

import (
	"reflect"
	"testing"
)

// Parsing is exercised end to end through Search, but a bug there only shows up
// as a wrong result set. These cases pin the shape the parser hands over.
func TestParseQuery(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		wantTerms   []string
		wantPhrases [][]phraseTerm
	}{
		{
			name:      "no quotes",
			query:     "wireless printer",
			wantTerms: []string{"wireless", "printer"},
		},
		{
			name:      "whole query quoted",
			query:     `"wireless printer"`,
			wantTerms: []string{"wireless", "printer"},
			wantPhrases: [][]phraseTerm{
				{{word: "wireless", offset: 0}, {word: "printer", offset: 1}},
			},
		},
		{
			name:      "phrase and free terms keep query order",
			query:     `canon "wireless printer" laser`,
			wantTerms: []string{"canon", "wireless", "printer", "laser"},
			wantPhrases: [][]phraseTerm{
				{{word: "wireless", offset: 0}, {word: "printer", offset: 1}},
			},
		},
		{
			// A stop word inside the quotes widens the offset instead of
			// closing the gap, so the document has to reproduce it.
			name:      "stop words widen the offsets",
			query:     `"imprimante de bureau"`,
			wantTerms: []string{"imprimante", "bureau"},
			wantPhrases: [][]phraseTerm{
				{{word: "imprimante", offset: 0}, {word: "bureau", offset: 2}},
			},
		},
		{
			// Offsets are relative to the first surviving term, not to the
			// start of the quoted text.
			name:      "leading stop word does not shift the phrase",
			query:     `"la imprimante bureau"`,
			wantTerms: []string{"imprimante", "bureau"},
			wantPhrases: [][]phraseTerm{
				{{word: "imprimante", offset: 0}, {word: "bureau", offset: 1}},
			},
		},
		{
			name:      "several phrases",
			query:     `"wireless printer" "duplex printing"`,
			wantTerms: []string{"wireless", "printer", "duplex", "printing"},
			wantPhrases: [][]phraseTerm{
				{{word: "wireless", offset: 0}, {word: "printer", offset: 1}},
				{{word: "duplex", offset: 0}, {word: "printing", offset: 1}},
			},
		},
		{
			// The term is reported once, but it still constrains the phrase.
			name:      "a term repeated inside and outside the quotes",
			query:     `printer "wireless printer"`,
			wantTerms: []string{"printer", "wireless"},
			wantPhrases: [][]phraseTerm{
				{{word: "wireless", offset: 0}, {word: "printer", offset: 1}},
			},
		},
		{
			name:      "unclosed quote falls back to free terms",
			query:     `"wireless printer`,
			wantTerms: []string{"wireless", "printer"},
		},
		{
			name:      "closed phrase before an unclosed one",
			query:     `"wireless printer" canon "laser`,
			wantTerms: []string{"wireless", "printer", "canon", "laser"},
			wantPhrases: [][]phraseTerm{
				{{word: "wireless", offset: 0}, {word: "printer", offset: 1}},
			},
		},
		{
			name:      "quotes holding nothing indexable",
			query:     `"le de" printer`,
			wantTerms: []string{"printer"},
		},
		{
			name:      "empty quotes",
			query:     `"" printer`,
			wantTerms: []string{"printer"},
		},
		{
			// Punctuation is normalized inside the quotes like anywhere else,
			// so the phrase still lines up with what the indexer stored.
			name:      "punctuation and accents inside a phrase",
			query:     `"Imprimante, Réseau"`,
			wantTerms: []string{"imprimante", "reseau"},
			wantPhrases: [][]phraseTerm{
				{{word: "imprimante", offset: 0}, {word: "reseau", offset: 1}},
			},
		},
		{
			name:  "empty query",
			query: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parsed := parseQuery(test.query)

			if len(parsed.terms) != 0 || len(test.wantTerms) != 0 {
				if !reflect.DeepEqual(parsed.terms, test.wantTerms) {
					t.Errorf("terms = %v, want %v", parsed.terms, test.wantTerms)
				}
			}
			if len(parsed.phrases) != 0 || len(test.wantPhrases) != 0 {
				if !reflect.DeepEqual(parsed.phrases, test.wantPhrases) {
					t.Errorf("phrases = %v, want %v", parsed.phrases, test.wantPhrases)
				}
			}
		})
	}
}
