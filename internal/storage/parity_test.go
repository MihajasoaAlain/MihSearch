package storage

import (
	"reflect"
	"sort"
	"testing"

	"github.com/miih/miih-search/internal/models"
)

// The engine and the searcher are written against the Storage interface and are
// tested against the in-memory backend alone. That only proves anything if the
// two backends actually agree, so this runs one script of writes through both
// and compares everything the searcher would go on to read.

type comparablePosting struct {
	ExternalID string
	Frequency  int
	Positions  []int
	Field      string
}

type snapshot struct {
	Stats     models.IndexStats
	Documents map[string]models.Document
	Postings  map[string][]comparablePosting
	Lengths   map[string]map[string]int
}

var parityDocuments = []models.Document{
	{ID: "sku-1", Type: "product", Title: "HP Printer", Content: "wireless laser printer"},
	{ID: "sku-2", Type: "product", Title: "Canon Scanner", Content: "compact wireless scanner"},
	{ID: "sku-3", Type: "article", Title: "Choosing a printer", Content: "laser printer buying guide"},
}

var parityWords = []string{"printer", "wireless", "laser", "scanner", "absent"}

// fillIndex writes the same index through any backend, mimicking what the
// engine does, including the mutation paths: an upsert over an existing
// posting and a document cleared before being rewritten.
func fillIndex(t *testing.T, store Storage) map[string]int {
	t.Helper()

	ids := make(map[string]int, len(parityDocuments))
	for _, doc := range parityDocuments {
		ids[doc.ID] = mustSaveDocument(t, store, doc)
	}

	type write struct {
		externalID string
		word       string
		field      string
		frequency  int
		positions  []int
	}
	writes := []write{
		{"sku-1", "printer", models.FieldTitle, 1, []int{1}},
		{"sku-1", "printer", models.FieldContent, 1, []int{2}},
		{"sku-1", "wireless", models.FieldContent, 1, []int{0}},
		{"sku-1", "laser", models.FieldContent, 1, []int{1}},
		{"sku-2", "scanner", models.FieldTitle, 1, []int{1}},
		{"sku-2", "wireless", models.FieldContent, 1, []int{1}},
		{"sku-3", "printer", models.FieldTitle, 1, []int{2}},
		{"sku-3", "printer", models.FieldContent, 1, []int{1}},
		{"sku-3", "laser", models.FieldContent, 1, []int{0}},
		// Overwrites the row written three lines above: both backends must
		// update rather than ignore.
		{"sku-3", "printer", models.FieldContent, 2, []int{1, 4}},
	}
	for _, w := range writes {
		termID := mustSaveTerm(t, store, w.word)
		posting := models.Posting{Frequency: w.frequency, Positions: w.positions, Field: w.field}
		if err := store.SavePosting(posting, termID, ids[w.externalID]); err != nil {
			t.Fatalf("SavePosting(%q, %q) error = %v", w.word, w.field, err)
		}
	}

	lengths := map[string]map[string]int{
		"sku-1": {models.FieldTitle: 2, models.FieldContent: 3},
		"sku-2": {models.FieldTitle: 2, models.FieldContent: 3},
		"sku-3": {models.FieldTitle: 2, models.FieldContent: 4},
	}
	for externalID, fields := range lengths {
		for field, length := range fields {
			if err := store.SaveFieldLength(ids[externalID], field, length); err != nil {
				t.Fatalf("SaveFieldLength(%q, %q) error = %v", externalID, field, err)
			}
		}
	}

	// A document that gets re-indexed is cleared first; both backends must end
	// up in the same state afterwards.
	if err := store.DeletePostingsByDocument(ids["sku-2"]); err != nil {
		t.Fatalf("DeletePostingsByDocument() error = %v", err)
	}
	termID := mustSaveTerm(t, store, "scanner")
	posting := models.Posting{Frequency: 2, Positions: []int{1, 3}, Field: models.FieldContent}
	if err := store.SavePosting(posting, termID, ids["sku-2"]); err != nil {
		t.Fatalf("SavePosting() after clearing error = %v", err)
	}

	return ids
}

func takeSnapshot(t *testing.T, store Storage, ids map[string]int) snapshot {
	t.Helper()

	externalIDs := make(map[int]string, len(ids))
	documentIDs := make([]int, 0, len(ids))
	for externalID, id := range ids {
		externalIDs[id] = externalID
		documentIDs = append(documentIDs, id)
	}
	sort.Ints(documentIDs)

	stats, err := store.GetIndexStats()
	if err != nil {
		t.Fatalf("GetIndexStats() error = %v", err)
	}

	documents, err := store.GetDocumentsByIDs(documentIDs)
	if err != nil {
		t.Fatalf("GetDocumentsByIDs() error = %v", err)
	}
	byExternalID := make(map[string]models.Document, len(documents))
	for id, doc := range documents {
		// Internal ids are assigned by the backend and are not part of the
		// contract; the rest of the row is.
		doc.InternalID = 0
		byExternalID[externalIDs[id]] = doc
	}

	grouped, err := store.FindPostingsByTerms(parityWords)
	if err != nil {
		t.Fatalf("FindPostingsByTerms() error = %v", err)
	}
	postings := make(map[string][]comparablePosting, len(grouped))
	for word, list := range grouped {
		comparable := make([]comparablePosting, 0, len(list))
		for _, posting := range list {
			comparable = append(comparable, comparablePosting{
				ExternalID: externalIDs[posting.DocumentID],
				Frequency:  posting.Frequency,
				Positions:  posting.Positions,
				Field:      posting.Field,
			})
		}
		sort.Slice(comparable, func(i, j int) bool {
			if comparable[i].ExternalID != comparable[j].ExternalID {
				return comparable[i].ExternalID < comparable[j].ExternalID
			}
			return comparable[i].Field < comparable[j].Field
		})
		postings[word] = comparable
	}

	fieldLengths, err := store.GetFieldLengths(documentIDs)
	if err != nil {
		t.Fatalf("GetFieldLengths() error = %v", err)
	}
	lengths := make(map[string]map[string]int, len(fieldLengths))
	for id, fields := range fieldLengths {
		lengths[externalIDs[id]] = fields
	}

	return snapshot{Stats: stats, Documents: byExternalID, Postings: postings, Lengths: lengths}
}

func TestBackendsAgree(t *testing.T) {
	// Skips before touching the in-memory backend when no database is around,
	// so the comparison is never half-run.
	postgres := newPostgresTestStorage(t)
	memory := NewMemoryStorage()

	fromPostgres := takeSnapshot(t, postgres, fillIndex(t, postgres))
	fromMemory := takeSnapshot(t, memory, fillIndex(t, memory))

	if !reflect.DeepEqual(fromPostgres.Stats, fromMemory.Stats) {
		t.Errorf("stats differ:\n postgres = %+v\n memory   = %+v", fromPostgres.Stats, fromMemory.Stats)
	}
	if !reflect.DeepEqual(fromPostgres.Documents, fromMemory.Documents) {
		t.Errorf("documents differ:\n postgres = %+v\n memory   = %+v", fromPostgres.Documents, fromMemory.Documents)
	}
	if !reflect.DeepEqual(fromPostgres.Lengths, fromMemory.Lengths) {
		t.Errorf("field lengths differ:\n postgres = %+v\n memory   = %+v", fromPostgres.Lengths, fromMemory.Lengths)
	}
	for _, word := range parityWords {
		if !reflect.DeepEqual(fromPostgres.Postings[word], fromMemory.Postings[word]) {
			t.Errorf("postings for %q differ:\n postgres = %+v\n memory   = %+v",
				word, fromPostgres.Postings[word], fromMemory.Postings[word])
		}
	}
}
