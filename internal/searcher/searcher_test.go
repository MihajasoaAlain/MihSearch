package searcher

import (
	"reflect"
	"testing"

	"github.com/miih/miih-search/internal/engine"
	"github.com/miih/miih-search/internal/models"
	"github.com/miih/miih-search/internal/storage"
)

func newIndex(t *testing.T, documents ...models.Document) *storage.MemoryStorage {
	t.Helper()

	store := storage.NewMemoryStorage()
	search := engine.NewSearchEngine(store)
	for _, doc := range documents {
		if err := search.Index(doc); err != nil {
			t.Fatalf("Index(%q) error = %v", doc.ID, err)
		}
	}
	return store
}

func externalIDs(results []models.SearchResult) []string {
	ids := make([]string, 0, len(results))
	for _, result := range results {
		ids = append(ids, result.ExternalID)
	}
	return ids
}

func TestSearchRanksFullMatchesFirst(t *testing.T) {
	store := newIndex(t,
		models.Document{ID: "1", Title: "HP Printer", Content: "wireless laser printer with wifi"},
		models.Document{ID: "2", Title: "Canon Scanner", Content: "a wireless scanner"},
		models.Document{ID: "3", Title: "Ink cartridge", Content: "cartridge for an inkjet printer"},
	)

	results, err := NewSearcher(store).Search("wireless printer")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	// Document 1 matches both terms, the other two match a single one.
	if want := []string{"1", "2", "3"}; !reflect.DeepEqual(externalIDs(results), want) {
		t.Fatalf("ranking = %v, want %v", externalIDs(results), want)
	}
	if want := []string{"wireless", "printer"}; !reflect.DeepEqual(results[0].MatchedTerms, want) {
		t.Errorf("matched terms = %v, want %v", results[0].MatchedTerms, want)
	}
	if len(results[1].MatchedTerms) != 1 {
		t.Errorf("document 2 matched %v, want a single term", results[1].MatchedTerms)
	}
}

func TestSearchBoostsTitleMatches(t *testing.T) {
	store := newIndex(t,
		models.Document{ID: "content-match", Title: "Office supplies", Content: "printer"},
		models.Document{ID: "title-match", Title: "Printer", Content: "office supplies"},
	)

	results, err := NewSearcher(store).Search("printer")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
	if results[0].ExternalID != "title-match" {
		t.Errorf("ranking = %v, want the title match first", externalIDs(results))
	}
	if results[0].Score != 2 || results[1].Score != 1 {
		t.Errorf("scores = %v / %v, want the title boosted 2x", results[0].Score, results[1].Score)
	}
}

func TestSearchScoresFrequency(t *testing.T) {
	store := newIndex(t,
		models.Document{ID: "1", Title: "Guide", Content: "a laser printer beats an inkjet printer"},
		models.Document{ID: "2", Title: "Guide", Content: "a printer"},
	)

	results, err := NewSearcher(store).Search("printer")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if results[0].ExternalID != "1" || results[0].Score != 2 {
		t.Errorf("top result = %s with score %v, want document 1 with score 2",
			results[0].ExternalID, results[0].Score)
	}
}

func TestSearchReturnsMetadata(t *testing.T) {
	store := newIndex(t,
		models.Document{ID: "1", Type: "product", Title: "HP Printer", Content: "wireless laser printer"},
	)

	results, err := NewSearcher(store).Search("printer")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].Type != "product" || results[0].Title != "HP Printer" {
		t.Errorf("result = %+v, want the document metadata to be hydrated", results[0])
	}
}

func TestSearchEmptyQueries(t *testing.T) {
	store := newIndex(t,
		models.Document{ID: "1", Title: "HP Printer", Content: "wireless laser printer"},
	)
	s := NewSearcher(store)

	for _, query := range []string{"", "   ", "le de la"} {
		results, err := s.Search(query)
		if err != nil {
			t.Fatalf("Search(%q) error = %v", query, err)
		}
		if len(results) != 0 {
			t.Errorf("Search(%q) = %v, want no results", query, externalIDs(results))
		}
	}
}

func TestSearchUnknownTerm(t *testing.T) {
	store := newIndex(t,
		models.Document{ID: "1", Title: "HP Printer", Content: "wireless laser printer"},
	)

	results, err := NewSearcher(store).Search("photocopieur")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 0 {
		t.Errorf("got %v, want no results", externalIDs(results))
	}
}
