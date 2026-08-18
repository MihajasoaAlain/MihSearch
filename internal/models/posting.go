package models

// Indexable fields of a document. A term is indexed once per field, so the
// same word can carry a different weight depending on where it appears.
const (
	FieldTitle   = "title"
	FieldContent = "content"
)

type Posting struct {
	DocumentID int
	Frequency  int
	// Positions of the term inside its own field, not inside the document.
	Positions []int
	Field     string
}
