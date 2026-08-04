package models

type SearchResult struct {
	DocumentID string
	Frequency  int
	Field      string
	Score      float64
}
