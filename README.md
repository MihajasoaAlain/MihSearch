# MiihSearch

**MiihSearch** is a lightweight, embeddable search engine written in Go. It indexes documents through a configurable text-analysis pipeline, stores an inverted index in PostgreSQL, and aims to become a reusable search solution you can drop into any application — web apps, e-commerce platforms, SaaS products, CMSs, or internal tools — without relying on a heavyweight external search service.

> **Status:** early development. The core engine is complete: indexing, persistence, and multi-term, field-aware search ranked with BM25, including phrase queries, work today. See the [Roadmap](#roadmap).

---

## Features

* **Document indexing** — index structured documents (`ID`, `Type`, `Title`, `Content`)
* **Inverted index** — term → document postings with positions and frequencies
* **Text analysis pipeline** — tokenization, lowercasing, punctuation removal, accent removal, and stop-word filtering
* **Field-aware indexing** — title and content are indexed separately, and a title match weighs more than a content match
* **Multi-term search** — every term of the query is resolved in one round trip
* **BM25 ranking** — matches are weighed by how rare the term is and how long the field it sits in is, not by raw frequency
* **Phrase search** — quote part of a query to require those words side by side, in order, within one field
* **Idempotent indexing** — re-indexing a document refreshes it and drops the terms it no longer contains
* **Pluggable storage** — the engine and searcher depend on a `Storage` interface; PostgreSQL and in-memory backends ship with the project
* **PostgreSQL persistence** — durable storage via [pgx](https://github.com/jackc/pgx)
* **Automatic schema creation** — tables and indexes are created and migrated on startup, no manual migration needed

---

## Getting Started

### Prerequisites

* Go 1.26+
* A running PostgreSQL instance

### Setup

```bash
git clone https://github.com/miih/miih-search.git
cd miih-search
go mod download
```

Create a `.env` file at the project root with your database connection string:

```env
DATABASE_URL=postgres://user:password@localhost:5432/miihsearch
```

### Run

```bash
go run ./cmd/server                            # index the sample documents, then run a sample search
go run ./cmd/server index                      # index the sample documents
go run ./cmd/server search "wireless printer"    # search the index
go run ./cmd/server search '"wireless printer"'  # require the exact phrase
```

On startup, MiihSearch connects to PostgreSQL and creates or migrates the schema. With no argument it also indexes the sample documents defined in `cmd/server/main.go` and prints the results of a sample search:

```text
query: "wireless printer" — 3 result(s)
  1.36  [2] Canon Wireless Printer  (matched: wireless, printer)
  0.90  [1] HP Printer  (matched: wireless, printer)
  0.41  [3] Choosing a laser printer  (matched: printer)
```

### Test

The test suite runs against the in-memory backend, so no database is required:

```bash
go test ./...
```

---

## Usage

```go
package main

import (
    "github.com/miih/miih-search/internal/engine"
    "github.com/miih/miih-search/internal/models"
    "github.com/miih/miih-search/internal/searcher"
    "github.com/miih/miih-search/internal/storage"
)

func main() {
    // Connect to PostgreSQL (reads DATABASE_URL) and create the schema.
    db, err := storage.NewPostgresStorage()
    if err != nil {
        panic(err)
    }
    if err := db.CreateTables(); err != nil {
        panic(err)
    }

    // Create the engine and index a document.
    search := engine.NewSearchEngine(db)

    err = search.Index(models.Document{
        ID:      "1",
        Type:    "product",
        Title:   "HP Printer",
        Content: "wireless laser printer with Wi-Fi",
    })
    if err != nil {
        panic(err)
    }

    // Search the index. Results come back ranked, best first.
    s := searcher.NewSearcher(db)
    results, err := s.Search("wireless printer")
    if err != nil {
        panic(err)
    }

    // Quote a group of words to require them side by side, in order. Terms left
    // outside the quotes still rank the documents the phrase let through.
    results, err = s.Search(`"wireless printer" canon`)
    if err != nil {
        panic(err)
    }
    // results: []models.SearchResult{DocumentID, ExternalID, Type, Title,
    //                                Content, MatchedTerms, Score}
    _ = results
}
```

Swap `storage.NewPostgresStorage()` for `storage.NewMemoryStorage()` to run the same engine without a database — both satisfy the `storage.Storage` interface.

---

## How It Works

### Indexing

Every document passes through the analysis pipeline before being written to the inverted index:

```text
Document → Analyzer → Tokenizer → Normalized Terms → Inverted Index → PostgreSQL
```

The analyzer applies, in order: lowercasing, punctuation removal, accent removal, tokenization, and stop-word filtering. Dropped stop words do not shift the positions of the terms around them, so phrase and proximity queries stay possible later on.

Title and content are analyzed as two separate fields, and each distinct term produces one posting per field, carrying how many times it occurred and every position where it was found. Indexing a document that already exists refreshes it: its previous postings are discarded first, so a term removed from the new version stops matching it.

The index is persisted across four tables:

| Table       | Purpose                                                     |
| ----------- | ----------------------------------------------------------- |
| `documents` | Document metadata and content                               |
| `terms`     | Unique indexed words                                        |
| `postings`  | Links terms to documents, with frequencies, positions, and the field they appear in — unique per `(term, document, field)` |
| `document_fields` | Length of each indexed field, in terms — what BM25 normalizes against |

### Querying

The query goes through the same analyzer, so a search matches whatever the indexer stored. All of its terms are then resolved in a single query (`terms` joined with `postings`), and the matching documents are loaded in one more.

Ranking is BM25, applied per field and combined with the field weights, so a title match still counts twice a content match. Three things decide a score:

* **Rarity.** A term present in nearly every document says almost nothing about the ones it matches; a rare term is close to an identifier. This is the inverse document frequency, and it is what separates a real ranking from counting occurrences.
* **Saturation.** A term found ten times is worth more than a term found once, but not ten times more — the contribution flattens out as frequency grows (`k1 = 1.2`).
* **Field length.** The same match counts for more in a short title than in a long body (`b = 0.75`).

A document matching two common words can therefore legitimately rank below one matching a single rare word. The document frequency of each term comes straight from its posting list, so ranking costs no extra query.

### Phrase queries

Wrapping part of a query in double quotes turns it into a phrase: `"wireless printer" canon` only considers documents where *wireless* is immediately followed by *printer*. Quotes are read before the analyzer runs, since punctuation removal would otherwise erase them; everything left outside them stays an ordinary term and takes part in the ranking.

A phrase is a **filter**, not a boost. A document holding both words in the wrong order is not what was asked for, so it is dropped rather than ranked lower. Several quoted groups in one query all have to hold, and a phrase never spans the title and the content — the two are separate texts, and their positions cannot be read as one sequence.

The check runs entirely on the positions already carried by the postings the query fetched, so it costs no extra round trip. Stop words are handled on both sides at once: they keep their position when the indexer drops them, and the query's own gaps have to line up with the document's. `"printer of the year"` therefore matches a document containing *printer of the year* and not one containing *printer year*.

> Phrase search reads positions that indexing has always recorded, so unlike the BM25 change it needs **no re-indexing**.

> **Upgrading an existing index:** BM25 needs field lengths, which are recorded during indexing and cannot be reconstructed afterwards. An index built before this was added must be re-indexed — documents indexed without a recorded length simply skip length normalization until then.

---

## Architecture

```text
                 +----------------------+
                 |   Your Application   |
                 +----------+-----------+
                            |
                     MiihSearch SDK
                            |
            +---------------+---------------+
            |   Search Engine  |  Searcher  |
            |    (indexing)    |  (queries) |
            +---------------+---------------+
                            |
                 +----------+----------+
                 |                     |
             Analyzer         Storage interface
                 |                     |
          Tokenization         PostgreSQL / Memory
          Lowercasing
          Punctuation removal
          Accent removal
          Stop-word filtering
```

### Project Structure

```text
cmd/
    analyzer/     # Text-analysis pipeline (tokenizer, normalizers, stop words)
    server/       # Application entry point
internal/
    engine/       # Search engine: document indexing
    searcher/     # Query side: term lookup and result ranking
    models/       # Domain types: Document, Term, Posting, SearchResult
    storage/      # Storage interface, PostgreSQL and in-memory backends, schema migration
```

---

## Roadmap

### Phase 1 — Core Engine

* [x] Document indexing
* [x] Inverted index
* [x] Text analyzer
* [x] PostgreSQL persistence
* [x] Database-backed search (single term, frequency-scored)
* [x] Multi-term search
* [x] Field-aware indexing (title / content)

### Phase 2 — Search Quality

* [x] BM25 ranking — supersedes the separate TF-IDF step, which it subsumes
* [x] Phrase search
* [ ] Prefix search and autocomplete
* [ ] Fuzzy search
* [ ] Synonyms
* [ ] Result highlighting

### Phase 3 — Performance

* [ ] In-memory cache
* [ ] Parallel indexing
* [ ] Incremental indexing
* [ ] Batch indexing

### Phase 4 — Ecosystem

* [ ] REST API
* [ ] Docker image
* [ ] SDKs (Go, Java, Node.js)
* [ ] Distributed indexing

---

## Design Goals

MiihSearch is built to be **lightweight**, **fast**, and **framework-agnostic** — a modular engine that embeds into any project with minimal configuration. Each stage of the pipeline (analysis, indexing, storage) is a separate component, so individual pieces can be extended or swapped without touching the rest.

---

## Contributing

Contributions, feature requests, and bug reports are welcome. Open an issue to discuss an idea, or submit a pull request directly.

---

## License

This project is licensed under the [MIT License](LICENSE).
