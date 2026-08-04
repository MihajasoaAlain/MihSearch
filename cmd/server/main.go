package main

import (
	"fmt"

	"github.com/joho/godotenv"
	"github.com/miih/miih-search/internal/engine"
	"github.com/miih/miih-search/internal/models"
	searcher2 "github.com/miih/miih-search/internal/searcher"
	"github.com/miih/miih-search/internal/storage"
)

func main() {
	err := godotenv.Load()
	storage, err := storage.NewPostgresStorage()
	if err != nil {
		panic(err)
	}
	err = storage.CreateTables()
	if err != nil {
		panic(err)
	}
	fmt.Println("DATABASE ready")

	search := engine.NewSearchEngine(storage)

	search.Index(models.Document{
		ID:      "1",
		Type:    "product",
		Title:   "HP Printer",
		Content: "imprimante laser wifi couleur",
	})

	search.Index(models.Document{
		ID:      "2",
		Type:    "product",
		Title:   "Canon Printer",
		Content: "imprimante jet encre",
	})

	searcher := searcher2.NewSearcher(storage)
	results, err := searcher.Search("imprimante")
	fmt.Printf("%+v\n", results)
	//postings, err := storage.FindPostingByTerm("imprimante")
	//fmt.Printf("%+v\n", postings)
	//results := search.Search("imprimante")
	//fmt.Printf("%+v\n", results)
}
