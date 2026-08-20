package storage

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"sort"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
	"github.com/miih/miih-search/internal/models"
)

// Integration tests for the PostgreSQL backend. Everything the engine and the
// searcher rely on is exercised against a real database, because none of it is
// visible from the in-memory backend: an ON CONFLICT clause that ignores
// instead of overwriting, a column that does not bind, an array that does not
// survive the round trip.
//
// They run inside a throwaway schema, so pointing DATABASE_URL at a working
// database is safe: the real tables are never read or written.

const testSchema = "miihsearch_test"

// databaseURL reports the connection string to test against, and whether it was
// requested explicitly. An explicit request that cannot be honoured is a
// failure; falling back to a developer's .env and finding nothing is not.
func databaseURL() (url string, explicit bool) {
	if url := os.Getenv("MIIHSEARCH_TEST_DATABASE_URL"); url != "" {
		return url, true
	}
	// Loaded by hand: tests do not go through cmd/server, which is what
	// normally reads the .env file.
	_ = godotenv.Load("../../.env")
	return os.Getenv("DATABASE_URL"), false
}

func newPostgresTestStorage(t *testing.T) *PostgresStorage {
	t.Helper()

	url, explicit := databaseURL()
	if url == "" {
		t.Skip("no database configured: set MIIHSEARCH_TEST_DATABASE_URL to run the integration tests")
	}

	conn, err := pgx.Connect(context.Background(), url)
	if err != nil {
		if explicit {
			t.Fatalf("MIIHSEARCH_TEST_DATABASE_URL is set but unreachable: %v", err)
		}
		t.Skipf("no database reachable, skipping integration tests: %v", err)
	}

	setup := []string{
		fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE;", testSchema),
		fmt.Sprintf("CREATE SCHEMA %s;", testSchema),
		fmt.Sprintf("SET search_path TO %s;", testSchema),
	}
	for _, statement := range setup {
		if _, err := conn.Exec(context.Background(), statement); err != nil {
			conn.Close(context.Background())
			t.Fatalf("preparing the test schema: %v", err)
		}
	}

	store := &PostgresStorage{db: conn}
	if err := store.CreateTables(); err != nil {
		conn.Close(context.Background())
		t.Fatalf("CreateTables() error = %v", err)
	}

	t.Cleanup(func() {
		drop := fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE;", testSchema)
		if _, err := conn.Exec(context.Background(), drop); err != nil {
			t.Errorf("dropping the test schema: %v", err)
		}
		conn.Close(context.Background())
	})
	return store
}

func mustSaveDocument(t *testing.T, store Storage, doc models.Document) int {
	t.Helper()

	if err := store.SaveDocument(doc); err != nil {
		t.Fatalf("SaveDocument(%q) error = %v", doc.ID, err)
	}
	id, err := store.GetDocument(doc.ID)
	if err != nil {
		t.Fatalf("GetDocument(%q) error = %v", doc.ID, err)
	}
	return id
}

func mustSaveTerm(t *testing.T, store Storage, word string) int {
	t.Helper()

	if err := store.SaveTerm(models.Term{Word: word}); err != nil {
		t.Fatalf("SaveTerm(%q) error = %v", word, err)
	}
	id, err := store.GetTermID(word)
	if err != nil {
		t.Fatalf("GetTermID(%q) error = %v", word, err)
	}
	return id
}

func TestPostgresMigrationsAreIdempotent(t *testing.T) {
	store := newPostgresTestStorage(t)

	// CreateTables already ran once during setup; it runs on every startup, so
	// a second pass must be a no-op rather than an error.
	if err := store.CreateTables(); err != nil {
		t.Fatalf("second CreateTables() error = %v", err)
	}
}

func TestPostgresDocumentRoundTrip(t *testing.T) {
	store := newPostgresTestStorage(t)

	id := mustSaveDocument(t, store, models.Document{
		ID: "sku-1", Type: "product", Title: "HP Printer", Content: "wireless laser printer",
	})

	doc, err := store.GetDocumentByID(id)
	if err != nil {
		t.Fatalf("GetDocumentByID() error = %v", err)
	}
	want := models.Document{
		InternalID: id, ID: "sku-1", Type: "product",
		Title: "HP Printer", Content: "wireless laser printer",
	}
	if !reflect.DeepEqual(doc, want) {
		t.Errorf("document = %+v, want %+v", doc, want)
	}
}

func TestPostgresDocumentUpsertRefreshes(t *testing.T) {
	store := newPostgresTestStorage(t)

	id := mustSaveDocument(t, store, models.Document{ID: "sku-1", Title: "Old", Content: "old body"})
	again := mustSaveDocument(t, store, models.Document{ID: "sku-1", Title: "New", Content: "new body"})

	// Re-indexing must update the row in place, not insert a second one.
	if again != id {
		t.Fatalf("document id changed from %d to %d on re-save", id, again)
	}
	doc, err := store.GetDocumentByID(id)
	if err != nil {
		t.Fatalf("GetDocumentByID() error = %v", err)
	}
	if doc.Title != "New" || doc.Content != "new body" {
		t.Errorf("document = %+v, want the refreshed title and content", doc)
	}
}

func TestPostgresMissingRowsReportNotFound(t *testing.T) {
	store := newPostgresTestStorage(t)

	if _, err := store.GetDocument("does-not-exist"); err != ErrNotFound {
		t.Errorf("GetDocument() error = %v, want ErrNotFound", err)
	}
	if _, err := store.GetDocumentByID(4242); err != ErrNotFound {
		t.Errorf("GetDocumentByID() error = %v, want ErrNotFound", err)
	}
	if _, err := store.GetTermID("does-not-exist"); err == nil {
		t.Error("GetTermID() on an unknown word returned no error")
	}
}

func TestPostgresPostingRoundTrip(t *testing.T) {
	store := newPostgresTestStorage(t)

	documentID := mustSaveDocument(t, store, models.Document{ID: "sku-1", Title: "Printer"})
	termID := mustSaveTerm(t, store, "printer")

	posting := models.Posting{
		DocumentID: documentID,
		Frequency:  3,
		Positions:  []int{0, 4, 9},
		Field:      models.FieldContent,
	}
	if err := store.SavePosting(posting, termID, documentID); err != nil {
		t.Fatalf("SavePosting() error = %v", err)
	}

	got, err := store.FindPostingByTerm("printer")
	if err != nil {
		t.Fatalf("FindPostingByTerm() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d postings, want 1", len(got))
	}
	// The positions array is the part most likely to break in transit.
	if !reflect.DeepEqual(got[0], posting) {
		t.Errorf("posting = %+v, want %+v", got[0], posting)
	}
}

func TestPostgresPostingUpsertOverwrites(t *testing.T) {
	store := newPostgresTestStorage(t)

	documentID := mustSaveDocument(t, store, models.Document{ID: "sku-1", Title: "Printer"})
	termID := mustSaveTerm(t, store, "printer")

	first := models.Posting{DocumentID: documentID, Frequency: 1, Positions: []int{0}, Field: models.FieldContent}
	if err := store.SavePosting(first, termID, documentID); err != nil {
		t.Fatalf("SavePosting() error = %v", err)
	}
	second := models.Posting{DocumentID: documentID, Frequency: 4, Positions: []int{0, 2, 7, 11}, Field: models.FieldContent}
	if err := store.SavePosting(second, termID, documentID); err != nil {
		t.Fatalf("second SavePosting() error = %v", err)
	}

	got, err := store.FindPostingByTerm("printer")
	if err != nil {
		t.Fatalf("FindPostingByTerm() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d postings, want the conflict to update a single row", len(got))
	}
	// The original ON CONFLICT DO NOTHING silently kept the stale frequency,
	// which is exactly what made frequency scoring impossible.
	if !reflect.DeepEqual(got[0], second) {
		t.Errorf("posting = %+v, want the recomputed %+v", got[0], second)
	}
}

func TestPostgresPostingsCoexistPerField(t *testing.T) {
	store := newPostgresTestStorage(t)

	documentID := mustSaveDocument(t, store, models.Document{ID: "sku-1", Title: "Printer", Content: "printer"})
	termID := mustSaveTerm(t, store, "printer")

	for _, field := range []string{models.FieldTitle, models.FieldContent} {
		posting := models.Posting{DocumentID: documentID, Frequency: 1, Positions: []int{0}, Field: field}
		if err := store.SavePosting(posting, termID, documentID); err != nil {
			t.Fatalf("SavePosting(%q) error = %v", field, err)
		}
	}

	got, err := store.FindPostingByTerm("printer")
	if err != nil {
		t.Fatalf("FindPostingByTerm() error = %v", err)
	}
	// The uniqueness key includes the field, so one term in one document still
	// yields two rows.
	if len(got) != 2 {
		t.Fatalf("got %d postings, want one per field: %+v", len(got), got)
	}
	fields := []string{got[0].Field, got[1].Field}
	sort.Strings(fields)
	if want := []string{models.FieldContent, models.FieldTitle}; !reflect.DeepEqual(fields, want) {
		t.Errorf("fields = %v, want %v", fields, want)
	}
}

func TestPostgresDeletePostingsByDocument(t *testing.T) {
	store := newPostgresTestStorage(t)

	kept := mustSaveDocument(t, store, models.Document{ID: "kept", Title: "Printer"})
	cleared := mustSaveDocument(t, store, models.Document{ID: "cleared", Title: "Printer"})
	termID := mustSaveTerm(t, store, "printer")

	for _, documentID := range []int{kept, cleared} {
		posting := models.Posting{Frequency: 1, Positions: []int{0}, Field: models.FieldTitle}
		if err := store.SavePosting(posting, termID, documentID); err != nil {
			t.Fatalf("SavePosting() error = %v", err)
		}
	}

	if err := store.DeletePostingsByDocument(cleared); err != nil {
		t.Fatalf("DeletePostingsByDocument() error = %v", err)
	}

	got, err := store.FindPostingByTerm("printer")
	if err != nil {
		t.Fatalf("FindPostingByTerm() error = %v", err)
	}
	if len(got) != 1 || got[0].DocumentID != kept {
		t.Errorf("postings = %+v, want only the untouched document %d", got, kept)
	}
}

func TestPostgresFindPostingsByTerms(t *testing.T) {
	store := newPostgresTestStorage(t)

	documentID := mustSaveDocument(t, store, models.Document{ID: "sku-1", Title: "Wireless printer"})
	for _, word := range []string{"wireless", "printer"} {
		termID := mustSaveTerm(t, store, word)
		posting := models.Posting{Frequency: 1, Positions: []int{0}, Field: models.FieldTitle}
		if err := store.SavePosting(posting, termID, documentID); err != nil {
			t.Fatalf("SavePosting(%q) error = %v", word, err)
		}
	}

	grouped, err := store.FindPostingsByTerms([]string{"wireless", "printer", "absent"})
	if err != nil {
		t.Fatalf("FindPostingsByTerms() error = %v", err)
	}
	if len(grouped) != 2 {
		t.Fatalf("got %d groups, want 2 (the unknown term contributes none): %v", len(grouped), grouped)
	}
	for _, word := range []string{"wireless", "printer"} {
		if len(grouped[word]) != 1 {
			t.Errorf("group %q = %+v, want a single posting", word, grouped[word])
		}
	}

	empty, err := store.FindPostingsByTerms(nil)
	if err != nil {
		t.Fatalf("FindPostingsByTerms(nil) error = %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("FindPostingsByTerms(nil) = %v, want no groups", empty)
	}
}

func TestPostgresGetDocumentsByIDs(t *testing.T) {
	store := newPostgresTestStorage(t)

	first := mustSaveDocument(t, store, models.Document{ID: "sku-1", Title: "First"})
	second := mustSaveDocument(t, store, models.Document{ID: "sku-2", Title: "Second"})

	documents, err := store.GetDocumentsByIDs([]int{first, second, 4242})
	if err != nil {
		t.Fatalf("GetDocumentsByIDs() error = %v", err)
	}
	// An unknown id is absent from the result, not an error.
	if len(documents) != 2 {
		t.Fatalf("got %d documents, want 2: %+v", len(documents), documents)
	}
	if documents[first].Title != "First" || documents[second].Title != "Second" {
		t.Errorf("documents = %+v, want them keyed by internal id", documents)
	}

	empty, err := store.GetDocumentsByIDs(nil)
	if err != nil {
		t.Fatalf("GetDocumentsByIDs(nil) error = %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("GetDocumentsByIDs(nil) = %v, want no documents", empty)
	}
}

func TestPostgresFieldLengthsAndStats(t *testing.T) {
	store := newPostgresTestStorage(t)

	short := mustSaveDocument(t, store, models.Document{ID: "short", Title: "Printer"})
	long := mustSaveDocument(t, store, models.Document{ID: "long", Title: "Printer"})

	if err := store.SaveFieldLength(short, models.FieldContent, 2); err != nil {
		t.Fatalf("SaveFieldLength() error = %v", err)
	}
	if err := store.SaveFieldLength(long, models.FieldContent, 8); err != nil {
		t.Fatalf("SaveFieldLength() error = %v", err)
	}
	// Re-indexing recomputes the length, so a second write must overwrite.
	if err := store.SaveFieldLength(long, models.FieldContent, 6); err != nil {
		t.Fatalf("SaveFieldLength() upsert error = %v", err)
	}

	lengths, err := store.GetFieldLengths([]int{short, long})
	if err != nil {
		t.Fatalf("GetFieldLengths() error = %v", err)
	}
	if got := lengths[short][models.FieldContent]; got != 2 {
		t.Errorf("short content length = %d, want 2", got)
	}
	if got := lengths[long][models.FieldContent]; got != 6 {
		t.Errorf("long content length = %d, want the overwritten 6", got)
	}

	stats, err := store.GetIndexStats()
	if err != nil {
		t.Fatalf("GetIndexStats() error = %v", err)
	}
	if stats.DocumentCount != 2 {
		t.Errorf("document count = %d, want 2", stats.DocumentCount)
	}
	if got := stats.AverageFieldLength[models.FieldContent]; got != 4 {
		t.Errorf("average content length = %v, want 4", got)
	}
}

func TestPostgresIndexStatsOnEmptyIndex(t *testing.T) {
	store := newPostgresTestStorage(t)

	stats, err := store.GetIndexStats()
	if err != nil {
		t.Fatalf("GetIndexStats() on an empty index error = %v", err)
	}
	if stats.DocumentCount != 0 {
		t.Errorf("document count = %d, want 0", stats.DocumentCount)
	}
	if len(stats.AverageFieldLength) != 0 {
		t.Errorf("average lengths = %v, want none", stats.AverageFieldLength)
	}
}

func TestPostgresDeleteFieldLengthsByDocument(t *testing.T) {
	store := newPostgresTestStorage(t)

	documentID := mustSaveDocument(t, store, models.Document{ID: "sku-1", Title: "Printer"})
	if err := store.SaveFieldLength(documentID, models.FieldContent, 5); err != nil {
		t.Fatalf("SaveFieldLength() error = %v", err)
	}
	if err := store.DeleteFieldLengthsByDocument(documentID); err != nil {
		t.Fatalf("DeleteFieldLengthsByDocument() error = %v", err)
	}

	lengths, err := store.GetFieldLengths([]int{documentID})
	if err != nil {
		t.Fatalf("GetFieldLengths() error = %v", err)
	}
	if len(lengths) != 0 {
		t.Errorf("lengths = %v, want none after deletion", lengths)
	}
}
