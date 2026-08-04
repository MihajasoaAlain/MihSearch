# MiihSearch

**MiihSearch** is a lightweight, embeddable search engine written in Go. It indexes documents through a configurable text-analysis pipeline, stores an inverted index in PostgreSQL, and aims to become a reusable search solution you can drop into any application — web apps, e-commerce platforms, SaaS products, CMSs, or internal tools — without relying on a heavyweight external search service.

> **Status:** early development. Indexing, persistence, and database-backed single-term search work today. See the [Roadmap](#roadmap).

---

## Features

* **Document indexing** — index structured documents (`ID`, `Type`, `Title`, `Content`)
* **Inverted index** — term → document postings with positions and frequencies
* **Text analysis pipeline** — tokenization, lowercasing, punctuation removal, accent removal, and stop-word filtering
* **Database-backed search** — single-term search that queries the inverted index in PostgreSQL and returns frequency-scored results
* **Pluggable storage** — the engine and searcher depend on a `Storage` interface, so the PostgreSQL backend can be swapped out
* **PostgreSQL persistence** — durable storage via [pgx](https://github.com/jackc/pgx)
* **Automatic schema creation** — tables are created on startup, no manual migration needed

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
go run ./cmd/server
```

On startup, MiihSearch connects to PostgreSQL, creates the schema if it doesn't exist, indexes the sample documents defined in `cmd/server/main.go`, and runs a sample search whose results are printed to the console.

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

    // Search the index. Results are scored by term frequency.
    s := searcher.NewSearcher(db)
    results, err := s.Search("printer")
    if err != nil {
        panic(err)
    }
    // results: []models.SearchResult{DocumentID, Frequency, Field, Score}
    _ = results
}
```

---

## How It Works

Every document passes through the analysis pipeline before being written to the inverted index:

```text
Document → Analyzer → Tokenizer → Normalized Terms → Inverted Index → PostgreSQL
```

The analyzer applies, in order: tokenization, lowercasing, punctuation removal, accent removal, and stop-word filtering.

The index is persisted across three tables:

| Table       | Purpose                                                     |
| ----------- | ----------------------------------------------------------- |
| `documents` | Document metadata and content                               |
| `terms`     | Unique indexed words                                        |
| `postings`  | Links terms to documents, with frequencies, positions, and the field they appear in |

At query time, the `Searcher` looks a term up in PostgreSQL (`terms` joined with `postings`) and returns one `SearchResult` per matching document, currently scored by term frequency.

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
          Tokenization            PostgreSQL
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
    searcher/     # Query side: term lookup and result scoring
    models/       # Domain types: Document, Term, Posting, SearchResult
    repository/   # Data-access layer
    storage/      # Storage interface, PostgreSQL implementation, schema migration
```

---

## Roadmap

### Phase 1 — Core Engine

* [x] Document indexing
* [x] Inverted index
* [x] Text analyzer
* [x] PostgreSQL persistence
* [x] Database-backed search (single term, frequency-scored)
* [ ] Multi-term search
* [ ] Field-aware indexing (title / content)

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
