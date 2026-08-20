package searcher

import "math"

// BM25 parameters. k1 controls how fast term frequency saturates: a term found
// ten times is worth more than one found once, but not ten times more. b
// controls how strongly a long field is penalized, from 0 (not at all) to 1.
// These are the values BM25 is usually run with.
const (
	bm25K1 = 1.2
	bm25B  = 0.75
)

// idf weighs a term by how rare it is across the corpus. A word present in
// nearly every document says almost nothing about the ones it matches, while a
// rare word is close to an identifier.
//
// The 1 + ... form keeps the result positive even when a term appears in every
// document, where the textbook formula turns negative and would let a match
// lower a document's score.
func idf(documentCount int, documentFrequency int) float64 {
	if documentFrequency <= 0 || documentCount <= 0 {
		return 0
	}
	n := float64(documentCount)
	df := float64(documentFrequency)
	return math.Log(1 + (n-df+0.5)/(df+0.5))
}

// termScore is the BM25 saturation term: the raw frequency, damped by k1 and
// normalized by how long the field is compared to an average one. Matching
// twice in a ten-word title beats matching twice in a thousand-word body.
func termScore(frequency int, fieldLength int, averageFieldLength float64) float64 {
	if frequency <= 0 {
		return 0
	}
	f := float64(frequency)

	// Without a usable average there is nothing to compare the field against,
	// so fall back to no length normalization rather than dividing by zero.
	normalization := 1.0
	if averageFieldLength > 0 && fieldLength > 0 {
		normalization = 1 - bm25B + bm25B*(float64(fieldLength)/averageFieldLength)
	}

	return (f * (bm25K1 + 1)) / (f + bm25K1*normalization)
}
