package services

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"codeberg.org/chewrafa/archivist/internal/db"
	"codeberg.org/chewrafa/archivist/internal/models"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type ImportResult struct {
	Characters     int
	DLUsages       int
	Transactions   int
	CostOfLivings  int
	Registries     int
	Missions       int
	MissionEntries int
	Guilds         int
	Errors         []string
}

var weekHeaderRe = regexp.MustCompile(`(?i)^semana\s+del\s+(\d{1,2})\s+al\s+\d{1,2}\s+de\s+(\w+)\s+de\s+(\d{4})`)

var spanishMonths = map[string]time.Month{
	"enero":      time.January,
	"febrero":    time.February,
	"marzo":      time.March,
	"abril":      time.April,
	"mayo":       time.May,
	"junio":      time.June,
	"julio":      time.July,
	"agosto":     time.August,
	"septiembre": time.September,
	"octubre":    time.October,
	"noviembre":  time.November,
	"diciembre":  time.December,
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

func parseWeekHeader(s string) *time.Time {
	m := weekHeaderRe.FindStringSubmatch(s)
	if m == nil {
		return nil
	}
	day, _ := strconv.Atoi(m[1])
	monthName := strings.ToLower(m[2])
	year, _ := strconv.Atoi(m[3])
	month, ok := spanishMonths[monthName]
	if !ok {
		return nil
	}
	t := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	return &t
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
	return int(parseFloat(s))
}

func ImportExcel(f *excelize.File) ImportResult {
	var result ImportResult

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		importCharacters(tx, f, &result)
		importDLUsages(tx, f, &result)
		importTransactions(tx, f, &result)
		importCostOfLiving(tx, f, &result)
		importCharacterRegistry(tx, f, &result)
		importMissions(tx, f, &result)
		importGuilds(tx, f, &result)
		return nil
	})
	if err != nil {
		result.Errors = append(result.Errors, err.Error())
	}

	return result
}

func importCharacters(tx *gorm.DB, f *excelize.File, result *ImportResult) {
	sheet := "Lista de Personajes"
	rows, err := f.GetRows(sheet)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Error reading %s: %v", sheet, err))
		return
	}

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
		if len(row) > 10 {
			guildName = strings.TrimSpace(row[10])
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

		tx.Where("name = ?", character.Name).FirstOrCreate(&character)
		result.Characters++
	}
}

func importDLUsages(tx *gorm.DB, f *excelize.File, result *ImportResult) {
	sheet := "Uso de DL"
	rows, err := f.GetRows(sheet)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Error reading %s: %v", sheet, err))
		return
	}

	var lastDate *time.Time
	for i, row := range rows {
		if i == 0 || len(row) < 3 {
			continue
		}

		charName := strings.TrimSpace(row[1])
		if charName == "" {
			continue
		}

		var character models.Character
		if err := tx.Where("name = ?", charName).First(&character).Error; err != nil {
			continue
		}

		dateStr := strings.TrimSpace(row[0])
		if dateStr != "" {
			dt := parseDate(dateStr)
			if dt != nil {
				lastDate = dt
			} else {
				if wh := parseWeekHeader(dateStr); wh != nil {
					lastDate = wh
				}
			}
		}

		if lastDate == nil {
			continue
		}

		dlVal := parseInt(row[2])
		goldChange := float64(0)
		if len(row) > 3 {
			goldChange = parseFloat(row[3])
		}
		desc := ""
		if len(row) > 4 {
			desc = strings.TrimSpace(row[4])
		}
		usage := models.DLUsage{
			Date:        *lastDate,
			CharacterID: character.ID,
			DLUsed:      dlVal,
			GoldChange:  goldChange,
			Description: desc,
		}

		tx.Create(&usage)
		result.DLUsages++
	}
}

func importTransactions(tx *gorm.DB, f *excelize.File, result *ImportResult) {
	sheet := "Compras"
	rows, err := f.GetRows(sheet)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Error reading %s: %v", sheet, err))
		return
	}

	for i, row := range rows {
		if i == 0 || len(row) < 4 {
			continue
		}

		charName := strings.TrimSpace(row[1])
		if charName == "" {
			continue
		}

		var character models.Character
		if err := tx.Where("name = ?", charName).First(&character).Error; err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Character not found: %s", charName))
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

		txEntry := models.Transaction{
			Date:        *dt,
			CharacterID: character.ID,
			Amount:      parseFloat(row[2]),
			Notes:       strings.TrimSpace(row[3]),
		}

		tx.Create(&txEntry)
		result.Transactions++
	}
}

func importCostOfLiving(tx *gorm.DB, f *excelize.File, result *ImportResult) {
	sheet := "Costo de Vida"
	rows, err := f.GetRows(sheet)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Error reading %s: %v", sheet, err))
		return
	}

	if len(rows) < 2 {
		return
	}

	header := rows[0]
	var colDates []*time.Time
	for j := 3; j < len(header); j++ {
		dateStr := strings.TrimSpace(header[j])
		if dateStr == "" {
			colDates = append(colDates, nil)
		} else {
			colDates = append(colDates, parseDate(dateStr))
		}
	}

	for i := 1; i < len(rows); i++ {
		row := rows[i]
		if len(row) < 4 {
			continue
		}

		charName := strings.TrimSpace(row[0])
		if charName == "" {
			continue
		}

		var character models.Character
		if err := tx.Where("name = ?", charName).First(&character).Error; err != nil {
			continue
		}

		guildRole := strings.TrimSpace(row[1])
		mount := strings.TrimSpace(row[2])
		if guildRole != "" || mount != "" {
			updates := map[string]interface{}{}
			if guildRole != "" {
				updates["guild_role"] = guildRole
			}
			if mount != "" {
				updates["mount"] = mount
			}
			tx.Model(&character).Updates(updates)
		}

		for j := 3; j < len(row) && j-3 < len(colDates); j++ {
			amountStr := strings.TrimSpace(row[j])
			if amountStr == "" {
				continue
			}
			amount := parseFloat(amountStr)
			if amount == 0 {
				continue
			}
			dt := colDates[j-3]
			if dt == nil {
				continue
			}

			cost := models.CostOfLiving{
				Date:        *dt,
				CharacterID: character.ID,
				Amount:      amount,
			}

			tx.Create(&cost)
			result.CostOfLivings++
		}
	}
}

func importCharacterRegistry(tx *gorm.DB, f *excelize.File, result *ImportResult) {
	sheet := "Registro de Personajes"
	rows, err := f.GetRows(sheet)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Error reading %s: %v", sheet, err))
		return
	}

	for i, row := range rows {
		if i == 0 || len(row) < 5 {
			continue
		}

		charName := strings.TrimSpace(row[1])
		if charName == "" {
			continue
		}

		var character models.Character
		if err := tx.Where("name = ?", charName).First(&character).Error; err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Character not found: %s", charName))
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

		tx.Create(&registry)
		result.Registries++
	}
}

func importMissions(tx *gorm.DB, f *excelize.File, result *ImportResult) {
	sheet := "Registro de Misiones"
	rows, err := f.GetRows(sheet)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Error reading %s: %v", sheet, err))
		return
	}

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
				tx.Create(&mission)
				currentMission = &mission
				result.Missions++
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
		if err := tx.Where("name = ?", charName).First(&character).Error; err != nil {
			continue
		}

		entry := models.MissionEntry{
			MissionID:   currentMission.ID,
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

		tx.Create(&entry)
		result.MissionEntries++
	}
}

func importGuilds(tx *gorm.DB, f *excelize.File, result *ImportResult) {
	sheet := "Gremios"
	rows, err := f.GetRows(sheet)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Error reading %s: %v", sheet, err))
		return
	}

	guildMap := make(map[string]*models.Guild)

	for i, row := range rows {
		if i == 0 || len(row) < 3 {
			continue
		}

		guildName := strings.TrimSpace(row[2])
		memberName := ""
		if len(row) > 4 {
			memberName = strings.TrimSpace(row[4])
		}

		if guildName == "" && memberName == "" {
			continue
		}

		if guildName != "" {
			col := float64(0)
			if len(row) > 5 {
				col = parseFloat(row[5])
			}
			treasury := float64(0)
			if len(row) > 6 {
				treasury = parseFloat(row[6])
			}
			notes := ""
			if len(row) > 7 {
				notes = strings.TrimSpace(row[7])
			}

			guild := &models.Guild{
				Name:         guildName,
				CostOfLiving: col,
				Treasury:     treasury,
				Notes:        notes,
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
				if err := tx.Where("name = ?", leaderName).First(&leader).Error; err == nil {
					guild.LeaderID = &leader.ID
				}
			}

			tx.Where("name = ?", guild.Name).FirstOrCreate(guild)
			guildMap[guildName] = guild
			result.Guilds++
		}

		if memberName != "" {
			var member models.Character
			if err := tx.Where("name = ?", memberName).First(&member).Error; err == nil {
				for _, g := range guildMap {
					var existing int64
					tx.Raw("SELECT COUNT(*) FROM guild_members WHERE guild_id = ? AND character_id = ?", g.ID, member.ID).Scan(&existing)
					if existing == 0 {
						tx.Exec("INSERT INTO guild_members (guild_id, character_id) VALUES (?, ?)", g.ID, member.ID)
					}
				}
			}
		}
	}
}

func init() {
	log.SetPrefix("[importer] ")
}
