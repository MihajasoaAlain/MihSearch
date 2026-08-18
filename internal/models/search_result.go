package models

type SearchResult struct {
	DocumentID int
	ExternalID string
	Type       string
	Title      string
	Content    string
	// MatchedTerms lists the query terms this document actually matched.
	MatchedTerms []string
	Score        float64
}
