package storage

import (
	"errors"

	"github.com/miih/miih-search/internal/models"
)

// ErrNotFound is returned by a Storage implementation when a lookup matches no row.
var ErrNotFound = errors.New("storage: not found")

type Storage interface {
	SaveDocument(doc models.Document) error
	SaveTerm(term models.Term) error
	SavePosting(posting models.Posting, termID int, documentID int) error
	DeletePostingsByDocument(documentID int) error
	GetTermID(word string) (int, error)
	GetDocument(externalID string) (int, error)
	GetDocumentByID(id int) (models.Document, error)
	GetDocumentsByIDs(ids []int) (map[int]models.Document, error)
	FindPostingByTerm(word string) ([]models.Posting, error)
	FindPostingsByTerms(words []string) (map[string][]models.Posting, error)
}
