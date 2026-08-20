package engine

import (
	"reflect"
	"testing"

	"github.com/miih/miih-search/internal/models"
	"github.com/miih/miih-search/internal/storage"
)

func TestIndexSeparatesTitleAndContent(t *testing.T) {
	store := storage.NewMemoryStorage()

	err := NewSearchEngine(store).Index(models.Document{
		ID:      "1",
		Type:    "product",
		Title:   "Laser Printer",
		Content: "wireless laser printer",
	})
	if err != nil {
		t.Fatalf("Index() error = %v", err)
	}

	postings, err := store.FindPostingByTerm("printer")
	if err != nil {
		t.Fatalf("FindPostingByTerm() error = %v", err)
	}
	if len(postings) != 2 {
		t.Fatalf("got %d postings for %q, want one per field: %+v", len(postings), "printer", postings)
	}

	fields := map[string]models.Posting{}
	for _, posting := range postings {
		fields[posting.Field] = posting
	}
	if _, ok := fields[models.FieldTitle]; !ok {
		t.Errorf("no title posting, got fields %v", fields)
	}
	if _, ok := fields[models.FieldContent]; !ok {
		t.Errorf("no content posting, got fields %v", fields)
	}
	if got := fields[models.FieldContent].Positions; !reflect.DeepEqual(got, []int{2}) {
		t.Errorf("content positions = %v, want [2]", got)
	}
}

func TestIndexCountsRepeatedTerms(t *testing.T) {
	store := storage.NewMemoryStorage()

	err := NewSearchEngine(store).Index(models.Document{
		ID:      "1",
		Title:   "Printers",
		Content: "a laser printer is cheaper than an inkjet printer",
	})
	if err != nil {
		t.Fatalf("Index() error = %v", err)
	}

	postings, err := store.FindPostingByTerm("printer")
	if err != nil {
		t.Fatalf("FindPostingByTerm() error = %v", err)
	}
	if len(postings) != 1 {
		t.Fatalf("got %d postings, want 1: %+v", len(postings), postings)
	}
	if postings[0].Frequency != 2 {
		t.Errorf("frequency = %d, want 2", postings[0].Frequency)
	}
	if want := []int{2, 8}; !reflect.DeepEqual(postings[0].Positions, want) {
		t.Errorf("positions = %v, want %v", postings[0].Positions, want)
	}
}

func TestIndexIsIdempotentAndDropsStaleTerms(t *testing.T) {
	store := storage.NewMemoryStorage()
	search := NewSearchEngine(store)

	doc := models.Document{ID: "1", Title: "Printer", Content: "wireless laser printer"}
	if err := search.Index(doc); err != nil {
		t.Fatalf("Index() error = %v", err)
	}

	doc.Content = "wired inkjet printer"
	if err := search.Index(doc); err != nil {
		t.Fatalf("re-Index() error = %v", err)
	}

	stale, err := store.FindPostingByTerm("laser")
	if err != nil {
		t.Fatalf("FindPostingByTerm() error = %v", err)
	}
	if len(stale) != 0 {
		t.Errorf("term removed from the document still has postings: %+v", stale)
	}

	fresh, err := store.FindPostingByTerm("inkjet")
	if err != nil {
		t.Fatalf("FindPostingByTerm() error = %v", err)
	}
	if len(fresh) != 1 {
		t.Errorf("got %d postings for the new term, want 1", len(fresh))
	}
}

func TestIndexRecordsFieldLengths(t *testing.T) {
	store := storage.NewMemoryStorage()

	// "le" and "de" are stop words: the recorded length must count the terms
	// that reached the index, not the words of the original text.
	err := NewSearchEngine(store).Index(models.Document{
		ID:      "1",
		Title:   "Imprimante laser",
		Content: "le prix de la cartouche",
	})
	if err != nil {
		t.Fatalf("Index() error = %v", err)
	}

	lengths, err := store.GetFieldLengths([]int{1})
	if err != nil {
		t.Fatalf("GetFieldLengths() error = %v", err)
	}
	if got := lengths[1][models.FieldTitle]; got != 2 {
		t.Errorf("title length = %d, want 2", got)
	}
	if got := lengths[1][models.FieldContent]; got != 2 {
		t.Errorf("content length = %d, want 2", got)
	}

	stats, err := store.GetIndexStats()
	if err != nil {
		t.Fatalf("GetIndexStats() error = %v", err)
	}
	if stats.DocumentCount != 1 {
		t.Errorf("document count = %d, want 1", stats.DocumentCount)
	}
	if stats.AverageFieldLength[models.FieldContent] != 2 {
		t.Errorf("average content length = %v, want 2", stats.AverageFieldLength[models.FieldContent])
	}
}

func TestIndexRefreshesFieldLengths(t *testing.T) {
	store := storage.NewMemoryStorage()
	search := NewSearchEngine(store)

	doc := models.Document{ID: "1", Title: "Printer", Content: "wireless laser printer"}
	if err := search.Index(doc); err != nil {
		t.Fatalf("Index() error = %v", err)
	}

	doc.Content = "printer"
	if err := search.Index(doc); err != nil {
		t.Fatalf("re-Index() error = %v", err)
	}

	lengths, err := store.GetFieldLengths([]int{1})
	if err != nil {
		t.Fatalf("GetFieldLengths() error = %v", err)
	}
	if got := lengths[1][models.FieldContent]; got != 1 {
		t.Errorf("content length = %d after shrinking the document, want 1", got)
	}
}
