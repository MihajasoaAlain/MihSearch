package models

type Posting struct {
	DocumentID string
	Frequency  int
	Positions  []int
	Field      string
}
