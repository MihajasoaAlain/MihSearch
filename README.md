# MiihSearch

**MiihSearch** is a lightweight, embeddable search engine written in Go. It indexes documents through a configurable text-analysis pipeline, stores an inverted index in PostgreSQL, and aims to become a reusable search solution you can drop into any application — web apps, e-commerce platforms, SaaS products, CMSs, or internal tools — without relying on a heavyweight external search service.

> **Status:** early development. The core engine is complete: indexing, persistence, and multi-term, field-aware search work today. See the [Roadmap](#roadmap).

---

## Features

* **Document indexing** — index structured documents (`ID`, `Type`, `Title`, `Content`)
* **Inverted index** — term → document postings with positions and frequencies
* **Text analysis pipeline** — tokenization, lowercasing, punctuation removal, accent removal, and stop-word filtering
* **Field-aware indexing** — title and content are indexed separately, and a title match weighs more than a content match
* **Multi-term search** — every term of the query is resolved in one round trip; documents covering more of the query rank higher
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
go run ./cmd/server search "wireless printer"  # search the index
```

On startup, MiihSearch connects to PostgreSQL and creates or migrates the schema. With no argument it also indexes the sample documents defined in `cmd/server/main.go` and prints the results of a sample search:

```text
query: "wireless printer" — 3 result(s)
  5.00  [2] Canon Wireless Printer  (matched: wireless, printer)
  4.00  [1] HP Printer  (matched: wireless, printer)
  4.00  [3] Choosing a laser printer  (matched: printer)
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

The index is persisted across three tables:

| Table       | Purpose                                                     |
| ----------- | ----------------------------------------------------------- |
| `documents` | Document metadata and content                               |
| `terms`     | Unique indexed words                                        |
| `postings`  | Links terms to documents, with frequencies, positions, and the field they appear in — unique per `(term, document, field)` |

### Querying

The query goes through the same analyzer, so a search matches whatever the indexer stored. All of its terms are then resolved in a single query (`terms` joined with `postings`), and the matching documents are loaded in one more.

Each match contributes `frequency × field boost` to a document's score, where a title match is worth twice a content match. Results are ranked by how many distinct query terms they matched first, then by score — a document covering the whole query always outranks one that only matched a single common word.

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

* [ ] TF-IDF scoring
* [ ] BM25 ranking
* [ ] Phrase search
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
