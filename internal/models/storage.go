package models

type Storage struct {
	SaveDocument func(doc Document) error
}
