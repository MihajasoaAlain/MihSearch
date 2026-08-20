package searcher

import (
	"fmt"
	"math"
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
	// Both fields hold a single term, so the whole gap comes from the boost.
	if ratio := results[0].Score / results[1].Score; math.Abs(ratio-2) > 1e-9 {
		t.Errorf("title/content score ratio = %v, want 2", ratio)
	}
}

func TestSearchScoresFrequency(t *testing.T) {
	// Both contents hold five terms, so length normalization cancels out and
	// only the number of occurrences can separate them.
	store := newIndex(t,
		models.Document{ID: "twice", Title: "Guide", Content: "laser printer beats inkjet printer"},
		models.Document{ID: "once", Title: "Guide", Content: "laser printer beats inkjet scanner"},
	)

	results, err := NewSearcher(store).Search("printer")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
	if results[0].ExternalID != "twice" {
		t.Errorf("ranking = %v, want the document matching twice first", externalIDs(results))
	}
}

func TestSearchPenalizesLongFields(t *testing.T) {
	// Same term, same frequency, different field lengths: BM25 prefers the
	// document where the match makes up more of the text.
	store := newIndex(t,
		models.Document{ID: "short", Title: "Guide", Content: "wireless printer"},
		models.Document{ID: "long", Title: "Guide", Content: "wireless printer that also scans copies faxes and staples documents"},
	)

	results, err := NewSearcher(store).Search("printer")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if results[0].ExternalID != "short" {
		t.Errorf("ranking = %v, want the shorter document first", externalIDs(results))
	}
}

func TestSearchFavoursRareTerms(t *testing.T) {
	// A corpus where one word is everywhere and another appears once. Fields
	// are all the same length, so rarity is the only thing left to rank on.
	documents := make([]models.Document, 0, 30)
	for i := 0; i < 28; i++ {
		documents = append(documents, models.Document{
			ID:      fmt.Sprintf("filler-%02d", i),
			Title:   "Catalogue",
			Content: "printer standard model page",
		})
	}
	documents = append(documents,
		models.Document{ID: "common-heavy", Title: "Catalogue", Content: "printer printer printer page"},
		models.Document{ID: "rare-single", Title: "Catalogue", Content: "thermosublimation standard model page"},
	)
	store := newIndex(t, documents...)

	results, err := NewSearcher(store).Search("printer thermosublimation")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	// Under the previous frequency-only scoring, common-heavy won with three
	// matches. "printer" is in 29 of 30 documents and says almost nothing.
	if results[0].ExternalID != "rare-single" {
		t.Fatalf("top result = %s, want the document holding the rare term", results[0].ExternalID)
	}
	var heavy models.SearchResult
	for _, result := range results {
		if result.ExternalID == "common-heavy" {
			heavy = result
		}
	}
	if results[0].Score <= heavy.Score {
		t.Errorf("rare term scored %v, common term matched three times scored %v; rarity should dominate",
			results[0].Score, heavy.Score)
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
