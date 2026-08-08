package models

type Posting struct {
	DocumentID int
	Frequency  int
	Positions  []int
	Field      string
}
