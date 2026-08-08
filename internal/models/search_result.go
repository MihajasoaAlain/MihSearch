package models

type SearchResult struct {
	DocumentID int
	ExternalID string
	Type       string
	Title      string
	Content    string
	Score      float64
}
