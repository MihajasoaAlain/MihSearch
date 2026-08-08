package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/miih/miih-search/internal/searcher"
	"github.com/miih/miih-search/internal/storage"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env loaded:", err)
	}

	store, err := storage.NewPostgresStorage()
	if err != nil {
		panic(err)
	}
	err = store.CreateTables()
	if err != nil {
		panic(err)
	}
	fmt.Println("DATABASE ready")

	if len(os.Args) < 2 {
		fmt.Println("usage: server <query>")
		return
	}
	query := os.Args[1]

	search := searcher.NewSearcher(store)
	results, err := search.Searcher(query)
	if err != nil {
		panic(err)
	}
	for _, result := range results {
		fmt.Printf("%.2f  [%s] %s\n", result.Score, result.ExternalID, result.Title)
	}
}
