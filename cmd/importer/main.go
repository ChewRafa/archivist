package main

import (
	"fmt"
	"log"
	"os"

	"codeberg.org/chewrafa/archivist/internal/db"
	"codeberg.org/chewrafa/archivist/internal/services"
	"github.com/xuri/excelize/v2"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: importer <path-to-excel-file>")
	}

	db.Init("data/archivist.db")

	f, err := excelize.OpenFile(os.Args[1])
	if err != nil {
		log.Fatal("Failed to open Excel file: ", err)
	}
	defer f.Close()

	result := services.ImportExcel(f)

	fmt.Printf("Characters: %d\n", result.Characters)
	fmt.Printf("DL Usages: %d\n", result.DLUsages)
	fmt.Printf("Transactions: %d\n", result.Transactions)
	fmt.Printf("Cost of Livings: %d\n", result.CostOfLivings)
	fmt.Printf("Character Registries: %d\n", result.Registries)
	fmt.Printf("Missions: %d\n", result.Missions)
	fmt.Printf("Mission Entries: %d\n", result.MissionEntries)
	fmt.Printf("Guilds: %d\n", result.Guilds)

	if len(result.Errors) > 0 {
		fmt.Println("\nErrors:")
		for _, err := range result.Errors {
			fmt.Println(" -", err)
		}
	}

	fmt.Println("\nImport completed successfully!")
}
