package models

type SearchResult struct {
	Documents Document
	Score     float64
	Frequency int
	Positions []int
	Field     string
}
