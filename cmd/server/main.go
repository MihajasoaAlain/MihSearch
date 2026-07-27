package main

import (
	"fmt"

	"github.com/miih/miih-search/internal/engine"
	"github.com/miih/miih-search/internal/models"
)

func main() {
	search := engine.NewSearchEngine()

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
	results := search.Search("Imprimante")
	fmt.Println(results)
}
