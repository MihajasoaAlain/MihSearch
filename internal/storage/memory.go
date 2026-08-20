package storage

import (
	"sort"
	"sync"

	"github.com/miih/miih-search/internal/models"
)

// MemoryStorage keeps the whole index in memory. It implements the same Storage
// interface as the PostgreSQL backend, which makes it useful for tests, demos
// and small embedded indexes that do not need to survive a restart.
type MemoryStorage struct {
	mu sync.RWMutex

	documents    map[int]models.Document
	documentIDs  map[string]int
	termIDs      map[string]int
	postings     map[postingKey]models.Posting
	fieldLengths map[int]map[string]int

	nextDocumentID int
	nextTermID     int
}

type postingKey struct {
	termID     int
	documentID int
	field      string
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		documents:      make(map[int]models.Document),
		documentIDs:    make(map[string]int),
		termIDs:        make(map[string]int),
		postings:       make(map[postingKey]models.Posting),
		fieldLengths:   make(map[int]map[string]int),
		nextDocumentID: 1,
		nextTermID:     1,
	}
}

func (m *MemoryStorage) SaveDocument(doc models.Document) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	id, exists := m.documentIDs[doc.ID]
	if !exists {
		id = m.nextDocumentID
		m.nextDocumentID++
		m.documentIDs[doc.ID] = id
	}
	doc.InternalID = id
	m.documents[id] = doc
	return nil
}

func (m *MemoryStorage) SaveTerm(term models.Term) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.termIDs[term.Word]; !exists {
		m.termIDs[term.Word] = m.nextTermID
		m.nextTermID++
	}
	return nil
}

func (m *MemoryStorage) SavePosting(posting models.Posting, termID int, documentID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	posting.DocumentID = documentID
	m.postings[postingKey{termID: termID, documentID: documentID, field: posting.Field}] = posting
	return nil
}

func (m *MemoryStorage) DeletePostingsByDocument(documentID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for key := range m.postings {
		if key.documentID == documentID {
			delete(m.postings, key)
		}
	}
	return nil
}

func (m *MemoryStorage) SaveFieldLength(documentID int, field string, length int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.fieldLengths[documentID] == nil {
		m.fieldLengths[documentID] = make(map[string]int)
	}
	m.fieldLengths[documentID][field] = length
	return nil
}

func (m *MemoryStorage) DeleteFieldLengthsByDocument(documentID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.fieldLengths, documentID)
	return nil
}

func (m *MemoryStorage) GetFieldLengths(documentIDs []int) (map[int]map[string]int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	lengths := make(map[int]map[string]int, len(documentIDs))
	for _, documentID := range documentIDs {
		fields, exists := m.fieldLengths[documentID]
		if !exists {
			continue
		}
		copied := make(map[string]int, len(fields))
		for field, length := range fields {
			copied[field] = length
		}
		lengths[documentID] = copied
	}
	return lengths, nil
}

func (m *MemoryStorage) GetIndexStats() (models.IndexStats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	totals := make(map[string]int)
	counts := make(map[string]int)
	for _, fields := range m.fieldLengths {
		for field, length := range fields {
			totals[field] += length
			counts[field]++
		}
	}

	stats := models.IndexStats{
		DocumentCount:      len(m.documents),
		AverageFieldLength: make(map[string]float64, len(totals)),
	}
	for field, total := range totals {
		stats.AverageFieldLength[field] = float64(total) / float64(counts[field])
	}
	return stats, nil
}

func (m *MemoryStorage) GetTermID(word string) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	id, exists := m.termIDs[word]
	if !exists {
		return 0, ErrNotFound
	}
	return id, nil
}

func (m *MemoryStorage) GetDocument(externalID string) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	id, exists := m.documentIDs[externalID]
	if !exists {
		return 0, ErrNotFound
	}
	return id, nil
}

func (m *MemoryStorage) GetDocumentByID(id int) (models.Document, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	doc, exists := m.documents[id]
	if !exists {
		return models.Document{}, ErrNotFound
	}
	return doc, nil
}

func (m *MemoryStorage) GetDocumentsByIDs(ids []int) (map[int]models.Document, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	documents := make(map[int]models.Document, len(ids))
	for _, id := range ids {
		if doc, exists := m.documents[id]; exists {
			documents[id] = doc
		}
	}
	return documents, nil
}

func (m *MemoryStorage) FindPostingByTerm(word string) ([]models.Posting, error) {
	grouped, err := m.FindPostingsByTerms([]string{word})
	if err != nil {
		return nil, err
	}
	return grouped[word], nil
}

func (m *MemoryStorage) FindPostingsByTerms(words []string) (map[string][]models.Posting, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	grouped := make(map[string][]models.Posting, len(words))
	for _, word := range words {
		termID, exists := m.termIDs[word]
		if !exists {
			continue
		}
		var postings []models.Posting
		for key, posting := range m.postings {
			if key.termID == termID {
				postings = append(postings, posting)
			}
		}
		// Map iteration is random; callers expect a stable posting list.
		sort.Slice(postings, func(i, j int) bool {
			if postings[i].DocumentID != postings[j].DocumentID {
				return postings[i].DocumentID < postings[j].DocumentID
			}
			return postings[i].Field < postings[j].Field
		})
		if len(postings) > 0 {
			grouped[word] = postings
		}
	}
	return grouped, nil
}
