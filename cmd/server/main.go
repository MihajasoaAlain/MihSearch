package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/miih/miih-search/internal/engine"
	"github.com/miih/miih-search/internal/models"
	"github.com/miih/miih-search/internal/searcher"
	"github.com/miih/miih-search/internal/storage"
)

var sampleDocuments = []models.Document{
	{
		ID:      "1",
		Type:    "product",
		Title:   "HP Printer",
		Content: "wireless laser printer with Wi-Fi and duplex printing",
	},
	{
		ID:      "2",
		Type:    "product",
		Title:   "Canon Wireless Printer",
		Content: "compact inkjet printer for home offices",
	},
	{
		ID:      "3",
		Type:    "article",
		Title:   "Choosing a laser printer",
		Content: "a laser printer is cheaper to run than an inkjet printer",
	},
}

const usage = `usage:
  server                  index the sample documents, then run a sample search
  server index            index the sample documents
  server search <query>   search the index`

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env loaded:", err)
	}

	store, err := storage.NewPostgresStorage()
	if err != nil {
		log.Fatal(err)
	}
	if err := store.CreateTables(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("DATABASE ready")

	command := "demo"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	switch command {
	case "demo":
		if err := index(store); err != nil {
			log.Fatal(err)
		}
		if err := search(store, "wireless printer"); err != nil {
			log.Fatal(err)
		}
	case "index":
		if err := index(store); err != nil {
			log.Fatal(err)
		}
	case "search":
		if len(os.Args) < 3 {
			fmt.Println(usage)
			os.Exit(1)
		}
		if err := search(store, strings.Join(os.Args[2:], " ")); err != nil {
			log.Fatal(err)
		}
	default:
		fmt.Println(usage)
		os.Exit(1)
	}
}

func index(store storage.Storage) error {
	search := engine.NewSearchEngine(store)
	for _, doc := range sampleDocuments {
		if err := search.Index(doc); err != nil {
			return err
		}
	}
	fmt.Printf("indexed %d documents\n", len(sampleDocuments))
	return nil
}

func search(store storage.Storage, query string) error {
	results, err := searcher.NewSearcher(store).Search(query)
	if err != nil {
		return err
	}

	fmt.Printf("\nquery: %q — %d result(s)\n", query, len(results))
	for _, result := range results {
		fmt.Printf("%6.2f  [%s] %s  (matched: %s)\n",
			result.Score,
			result.ExternalID,
			result.Title,
			strings.Join(result.MatchedTerms, ", "),
		)
	}
	return nil
}
