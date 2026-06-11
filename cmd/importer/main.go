package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"codeberg.org/chewrafa/archivist/internal/db"
	"codeberg.org/chewrafa/archivist/internal/services"
	"github.com/xuri/excelize/v2"
)

func main() {
	sheetsFlag := flag.String("sheets", "", "Comma-separated list of sheets to import: characters,dlusages,transactions,costofliving,registry,missions,guilds,guildeconomy")
	flag.Parse()

	if flag.NArg() < 1 {
		log.Fatal("Usage: importer [--sheets=...] <path-to-excel-file>")
	}

	db.Init("data/archivist.db")

	f, err := excelize.OpenFile(flag.Arg(0))
	if err != nil {
		log.Fatal("Failed to open Excel file: ", err)
	}
	defer f.Close()

	var opts []services.ImportOptions
	if *sheetsFlag != "" {
		sheets := strings.Split(*sheetsFlag, ",")
		for i := range sheets {
			sheets[i] = strings.TrimSpace(sheets[i])
		}
		opts = append(opts, services.ImportOptions{Sheets: sheets})
	}

	result := services.ImportExcel(f, opts...)

	fmt.Printf("Characters: %d (skipped %d)\n", result.Characters, result.CharactersSkipped)
	fmt.Printf("DL Usages: %d (skipped %d)\n", result.DLUsages, result.DLUsagesSkipped)
	fmt.Printf("Transactions: %d (skipped %d)\n", result.Transactions, result.TransactionsSkipped)
	fmt.Printf("Cost of Livings: %d (skipped %d)\n", result.CostOfLivings, result.CostOfLivingsSkipped)
	fmt.Printf("Character Registries: %d (skipped %d)\n", result.Registries, result.RegistriesSkipped)
	fmt.Printf("Missions: %d (skipped %d)\n", result.Missions, result.MissionsSkipped)
	fmt.Printf("Mission Entries: %d (skipped %d)\n", result.MissionEntries, result.MissionEntriesSkipped)
	fmt.Printf("Guilds: %d (skipped %d)\n", result.Guilds, result.GuildsSkipped)
	fmt.Printf("Guild Transactions: %d (skipped %d)\n", result.GuildTransactions, result.GuildTransactionsSkipped)

	if len(result.Errors) > 0 {
		fmt.Println("\nErrors:")
		for _, err := range result.Errors {
			fmt.Println(" -", err)
		}
	}

	fmt.Println("\nImport completed successfully!")
}
