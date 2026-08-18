package models

// IndexStats holds the corpus-wide numbers BM25 needs to weigh a match: how
// many documents exist, and how long an average field is. They change only
// when documents are indexed, not between queries.
type IndexStats struct {
	DocumentCount int
	// AverageFieldLength is the mean number of indexed terms per field, keyed
	// by field name.
	AverageFieldLength map[string]float64
}
