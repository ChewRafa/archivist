package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"codeberg.org/chewrafa/archivist/internal/db"
	"codeberg.org/chewrafa/archivist/internal/models"
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

	importCharacters(f)
	importDLUsages(f)
	importTransactions(f)
	importCostOfLiving(f)
	importCharacterRegistry(f)
	importMissions(f)
	importGuilds(f)

	fmt.Println("Import completed successfully!")
}

func parseDate(s string) *time.Time {
	if s == "" {
		return nil
	}
	formats := []string{
		"2-1-06",
		"02-01-06",
		"2/1/06",
		"02/01/06",
		"2/1/2006",
		"02/01/2006",
		"2006-01-02",
		"2006-01-02 15:04:05",
	}
	for _, fmt := range formats {
		t, err := time.Parse(fmt, s)
		if err == nil {
			return &t
		}
	}
	return nil
}

func parseFloat(s string) float64 {
	if s == "" {
		return 0
	}
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "=", "")
	s = strings.ReplaceAll(s, "'", "")
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}

func parseInt(s string) int {
	if s == "" {
		return 0
	}
	v, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0
	}
	return v
}

func getCellString(f *excelize.File, sheet, cell string) string {
	v, err := f.GetCellValue(sheet, cell)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(v)
}

func importCharacters(f *excelize.File) {
	sheet := "Lista de Personajes"
	rows, err := f.GetRows(sheet)
	if err != nil {
		log.Printf("Error reading %s: %v", sheet, err)
		return
	}

	count := 0
	for i, row := range rows {
		if i == 0 || len(row) < 6 {
			continue
		}

		name := strings.TrimSpace(row[2])
		if name == "" {
			continue
		}

		status := strings.TrimSpace(row[3])
		if status == "" {
			status = "Activo"
		}

		guildName := ""
		if len(row) > 11 {
			guildName = strings.TrimSpace(row[11])
		}

		character := models.Character{
			Number:    parseInt(row[0]),
			Player:    strings.TrimSpace(row[1]),
			Name:      name,
			Status:    status,
			Species:   strings.TrimSpace(row[4]),
			Class:     strings.TrimSpace(row[5]),
			GuildName: guildName,
		}

		db.DB.Where("name = ?", character.Name).FirstOrCreate(&character)
		count++
	}
	fmt.Printf("Imported %d characters\n", count)
}

func importDLUsages(f *excelize.File) {
	sheet := "Uso de DL"
	rows, err := f.GetRows(sheet)
	if err != nil {
		log.Printf("Error reading %s: %v", sheet, err)
		return
	}

	count := 0
	for i, row := range rows {
		if i == 0 || len(row) < 5 {
			continue
		}

		charName := strings.TrimSpace(row[1])
		if charName == "" {
			continue
		}

		var character models.Character
		if err := db.DB.Where("name = ?", charName).First(&character).Error; err != nil {
			log.Printf("Character not found: %s", charName)
			continue
		}

		dateStr := strings.TrimSpace(row[0])
		if dateStr == "" {
			continue
		}

		dt := parseDate(dateStr)
		if dt == nil {
			continue
		}

		usage := models.DLUsage{
			Date:        *dt,
			CharacterID: character.ID,
			DLUsed:      max(parseInt(row[2]), -parseInt(row[2])),
			GoldChange:  parseFloat(row[3]),
			Description: strings.TrimSpace(row[4]),
		}

		db.DB.Create(&usage)
		count++
	}
	fmt.Printf("Imported %d DL usages\n", count)
}

func importTransactions(f *excelize.File) {
	sheet := "Compras"
	rows, err := f.GetRows(sheet)
	if err != nil {
		log.Printf("Error reading %s: %v", sheet, err)
		return
	}

	count := 0
	for i, row := range rows {
		if i == 0 || len(row) < 4 {
			continue
		}

		charName := strings.TrimSpace(row[1])
		if charName == "" {
			continue
		}

		var character models.Character
		if err := db.DB.Where("name = ?", charName).First(&character).Error; err != nil {
			log.Printf("Character not found: %s", charName)
			continue
		}

		dateStr := strings.TrimSpace(row[0])
		if dateStr == "" {
			continue
		}

		dt := parseDate(dateStr)
		if dt == nil {
			continue
		}

		tx := models.Transaction{
			Date:        *dt,
			CharacterID: character.ID,
			Amount:      parseFloat(row[2]),
			Notes:       strings.TrimSpace(row[3]),
		}

		db.DB.Create(&tx)
		count++
	}
	fmt.Printf("Imported %d transactions\n", count)
}

func importCostOfLiving(f *excelize.File) {
	sheet := "Costo de Vida"
	rows, err := f.GetRows(sheet)
	if err != nil {
		log.Printf("Error reading %s: %v", sheet, err)
		return
	}

	count := 0
	for i, row := range rows {
		if i == 0 || len(row) < 1 {
			continue
		}

		dateStr := strings.TrimSpace(row[0])
		if dateStr == "" {
			continue
		}

		dt := parseDate(dateStr)
		if dt == nil {
			continue
		}

		cost := models.CostOfLiving{
			Date: *dt,
		}
		if len(row) > 1 {
			cost.Inn = parseFloat(row[1])
		}
		if len(row) > 2 {
			cost.Guardiana = parseFloat(row[2])
		}
		if len(row) > 3 {
			cost.PlumaNegra = parseFloat(row[3])
		}
		if len(row) > 4 {
			cost.HijosAlba = parseFloat(row[4])
		}

		db.DB.Create(&cost)
		count++
	}
	fmt.Printf("Imported %d cost of living records\n", count)
}

func importCharacterRegistry(f *excelize.File) {
	sheet := "Registro de Personajes"
	rows, err := f.GetRows(sheet)
	if err != nil {
		log.Printf("Error reading %s: %v", sheet, err)
		return
	}

	count := 0
	for i, row := range rows {
		if i == 0 || len(row) < 5 {
			continue
		}

		charName := strings.TrimSpace(row[1])
		if charName == "" {
			continue
		}

		var character models.Character
		if err := db.DB.Where("name = ?", charName).First(&character).Error; err != nil {
			log.Printf("Character not found: %s", charName)
			continue
		}

		dateStr := strings.TrimSpace(row[0])
		if dateStr == "" {
			continue
		}

		dt := parseDate(dateStr)
		if dt == nil {
			continue
		}

		registry := models.CharacterRegistry{
			Date:        *dt,
			CharacterID: character.ID,
			Event:       strings.TrimSpace(row[2]),
			Experience:  parseFloat(row[3]),
			Gold:        parseFloat(row[4]),
		}
		if len(row) > 5 {
			registry.Renown = parseFloat(row[5])
		}
		if len(row) > 6 {
			registry.Notes = strings.TrimSpace(row[6])
		}

		db.DB.Create(&registry)
		count++
	}
	fmt.Printf("Imported %d character registry records\n", count)
}

func importMissions(f *excelize.File) {
	sheet := "Registro de Misiones"
	rows, err := f.GetRows(sheet)
	if err != nil {
		log.Printf("Error reading %s: %v", sheet, err)
		return
	}

	count := 0
	var currentMission *models.Mission

	for i, row := range rows {
		if i == 0 || len(row) < 3 {
			continue
		}

		dateStr := strings.TrimSpace(row[0])
		dm := ""
		if len(row) > 1 {
			dm = strings.TrimSpace(row[1])
		}
		eventName := ""
		if len(row) > 3 {
			eventName = strings.TrimSpace(row[3])
		}

		if dateStr != "" && dm != "" {
			dt := parseDate(dateStr)
			if dt != nil {
				mission := models.Mission{
					Date: *dt,
					DM:   dm,
					Name: eventName,
				}
				db.DB.Create(&mission)
				currentMission = &mission
			}
		}

		if currentMission == nil {
			continue
		}

		charName := strings.TrimSpace(row[2])
		if charName == "" {
			continue
		}

		var character models.Character
		if err := db.DB.Where("name = ?", charName).First(&character).Error; err != nil {
			continue
		}

		entry := models.MissionEntry{
			MissionID: currentMission.ID,
			CharacterID: character.ID,
		}
		if len(row) > 4 {
			entry.XPMission = parseFloat(row[4])
		}
		if len(row) > 5 {
			entry.XPReport = parseFloat(row[5])
		}
		if len(row) > 6 {
			entry.XPGuild = parseFloat(row[6])
		}
		if len(row) > 8 {
			entry.Gold = parseFloat(row[8])
		}
		if len(row) > 9 {
			entry.Renown = parseFloat(row[9])
		}
		if len(row) > 10 {
			entry.Notes = strings.TrimSpace(row[10])
		}

		db.DB.Create(&entry)
		count++
	}
	fmt.Printf("Imported %d mission entries\n", count)
}

func importGuilds(f *excelize.File) {
	sheet := "Gremios"
	rows, err := f.GetRows(sheet)
	if err != nil {
		log.Printf("Error reading %s: %v", sheet, err)
		return
	}

	guildMap := make(map[string]*models.Guild)
	count := 0

	for i, row := range rows {
		if i == 0 || len(row) < 9 {
			continue
		}

		guildName := strings.TrimSpace(row[2])
		memberName := strings.TrimSpace(row[4])

		if guildName == "" && memberName == "" {
			continue
		}

		if guildName != "" {
			guild := &models.Guild{
				Name:         guildName,
				Leader:       nil,
				HallType:     strings.TrimSpace(row[5]),
				Notes:        strings.TrimSpace(row[6]),
				CostOfLiving: parseFloat(row[7]),
				Treasury:     parseFloat(row[8]),
			}

			regDate := parseDate(strings.TrimSpace(row[0]))
			if regDate != nil {
				guild.RegisteredAt = regDate
			}

			appDate := parseDate(strings.TrimSpace(row[1]))
			if appDate != nil {
				guild.ApprovedAt = appDate
			}

			leaderName := strings.TrimSpace(row[3])
			if leaderName != "" {
				var leader models.Character
				if err := db.DB.Where("name = ?", leaderName).First(&leader).Error; err == nil {
					guild.LeaderID = &leader.ID
				}
			}

			db.DB.Where("name = ?", guild.Name).FirstOrCreate(guild)
			guildMap[guildName] = guild
			count++
		}

		if memberName != "" {
			var member models.Character
			if err := db.DB.Where("name = ?", memberName).First(&member).Error; err == nil {
				for _, g := range guildMap {
					var existing uint
					db.DB.Raw("SELECT COUNT(*) FROM guild_members WHERE guild_id = ? AND character_id = ?", g.ID, member.ID).Scan(&existing)
					if existing == 0 {
						db.DB.Exec("INSERT INTO guild_members (guild_id, character_id) VALUES (?, ?)", g.ID, member.ID)
					}
				}
			}
		}
	}
	fmt.Printf("Imported %d guilds\n", count)
}
