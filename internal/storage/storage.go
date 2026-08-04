package storage

import "github.com/miih/miih-search/internal/models"

type Storage interface {
	SaveDocument(doc models.Document) error
	SaveTerm(term models.Term) error
	SavePosting(posting models.Posting, termID int, documentID int) error
	GetTermID(word string) (int, error)
	GetDocument(externalID string) (int, error)
}
