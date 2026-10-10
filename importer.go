package main

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type ImportResult struct {
	Characters               int
	CharactersSkipped        int
	DLUsages                 int
	DLUsagesSkipped          int
	Transactions             int
	TransactionsSkipped      int
	CostOfLivings            int
	CostOfLivingsSkipped     int
	Registries               int
	RegistriesSkipped        int
	Missions                 int
	MissionsSkipped          int
	MissionEntries           int
	MissionEntriesSkipped    int
	Guilds                   int
	GuildsSkipped            int
	GuildTransactions        int
	GuildTransactionsSkipped int
	GuildMembers             int
	Errors                   []string
}

type ImportOptions struct {
	Sheets []string
}

var sheetKeyToName = map[string]string{
	"characters":   "Lista de Personajes",
	"dlusages":     "Uso de DL",
	"transactions": "Compras",
	"costofliving": "Costo de Vida",
	"registry":     "Registro de Personajes",
	"missions":     "Registro de Misiones",
	"guilds":       "Gremios",
	"guildeconomy": "Economía de Gremios",
}

var allSheetKeys = func() []string {
	keys := make([]string, 0, len(sheetKeyToName))
	for k := range sheetKeyToName {
		keys = append(keys, k)
	}
	return keys
}()

type SheetInfo struct {
	Key  string
	Name string
}

var allSheetInfo = func() []SheetInfo {
	return []SheetInfo{
		{Key: "characters", Name: sheetKeyToName["characters"]},
		{Key: "dlusages", Name: sheetKeyToName["dlusages"]},
		{Key: "transactions", Name: sheetKeyToName["transactions"]},
		{Key: "costofliving", Name: sheetKeyToName["costofliving"]},
		{Key: "registry", Name: sheetKeyToName["registry"]},
		{Key: "missions", Name: sheetKeyToName["missions"]},
		{Key: "guilds", Name: sheetKeyToName["guilds"]},
		{Key: "guildeconomy", Name: sheetKeyToName["guildeconomy"]},
	}
}()

func AllSheetKeys() []string {
	return allSheetKeys
}

func AllSheetInfo() []SheetInfo {
	return allSheetInfo
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

func parseExcelDate(s string) *time.Time {
	if s == "" {
		return nil
	}
	if dt := parseDate(s); dt != nil {
		return dt
	}
	if v, err := strconv.ParseFloat(strings.TrimSpace(s), 64); err == nil {
		t, err := excelize.ExcelDateToTime(v, false)
		if err == nil {
			return &t
		}
	}
	return nil
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
	s = strings.ReplaceAll(s, ",", "")
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}

func parseInt(s string) int {
	return int(parseFloat(s))
}

func shouldImportSheet(opts *ImportOptions, key string) bool {
	if opts == nil || len(opts.Sheets) == 0 {
		return true
	}
	for _, s := range opts.Sheets {
		if s == key {
			return true
		}
	}
	return false
}

// withImportSavepoint runs fn inside a savepoint so a failed statement does not
// abort the surrounding import transaction. PostgreSQL rejects every later
// command after the first error; rolling back to the savepoint clears that.
func withImportSavepoint(tx *gorm.DB, fn func(tx *gorm.DB) error) error {
	const name = "archivist_import_row"
	if err := tx.Exec("SAVEPOINT " + name).Error; err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		if rbErr := tx.Exec("ROLLBACK TO SAVEPOINT " + name).Error; rbErr != nil {
			return fmt.Errorf("%w (rollback savepoint: %v)", err, rbErr)
		}
		if relErr := tx.Exec("RELEASE SAVEPOINT " + name).Error; relErr != nil {
			return fmt.Errorf("%w (release savepoint: %v)", err, relErr)
		}
		return err
	}
	return tx.Exec("RELEASE SAVEPOINT " + name).Error
}

func ImportExcel(f *excelize.File, opts ...ImportOptions) ImportResult {
	var result ImportResult

	var opt *ImportOptions
	if len(opts) > 0 {
		opt = &opts[0]
	}

	err := DB.Transaction(func(tx *gorm.DB) error {
		if shouldImportSheet(opt, "characters") {
			importCharacters(tx, f, &result)
		}
		if shouldImportSheet(opt, "dlusages") {
			importDLUsages(tx, f, &result)
		}
		if shouldImportSheet(opt, "transactions") {
			importTransactions(tx, f, &result)
		}
		if shouldImportSheet(opt, "costofliving") {
			importCostOfLiving(tx, f, &result)
		}
		if shouldImportSheet(opt, "registry") {
			importCharacterRegistry(tx, f, &result)
		}
		if shouldImportSheet(opt, "missions") {
			importMissions(tx, f, &result)
		}
		if shouldImportSheet(opt, "guilds") {
			importGuilds(tx, f, &result)
		}
		if shouldImportSheet(opt, "guildeconomy") {
			importGuildEconomy(tx, f, &result)
		}
		if err := RefreshCharacterStatsView(tx); err != nil {
			return err
		}
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

		character := Character{
			Number:    parseInt(row[0]),
			Player:    strings.TrimSpace(row[1]),
			Name:      name,
			Status:    status,
			Species:   strings.TrimSpace(row[4]),
			Class:     strings.TrimSpace(row[5]),
			GuildName: guildName,
		}

		var existing Character
		err := tx.Where("name = ?", character.Name).First(&existing).Error
		if err == nil {
			result.CharactersSkipped++
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			result.Errors = append(result.Errors, fmt.Sprintf("Error importing character %s: %v", character.Name, err))
			continue
		}
		if err := withImportSavepoint(tx, func(tx *gorm.DB) error {
			return tx.Create(&character).Error
		}); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Error importing character %s: %v", character.Name, err))
			continue
		}
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

		var character Character
		if err := tx.Where("name = ?", charName).First(&character).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				result.Errors = append(result.Errors, fmt.Sprintf("Error importing DL usage row %d: %v", i+1, err))
			}
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
		usage := DLUsage{
			Date:        *lastDate,
			CharacterID: character.ID,
			DLUsed:      dlVal,
			GoldChange:  goldChange,
			Description: desc,
		}

		var existing DLUsage
		err := tx.Where("date = ? AND character_id = ? AND dl_used = ? AND description = ?",
			usage.Date, usage.CharacterID, usage.DLUsed, usage.Description).First(&existing).Error
		if err == nil {
			result.DLUsagesSkipped++
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			result.Errors = append(result.Errors, fmt.Sprintf("Error importing DL usage row %d: %v", i+1, err))
			continue
		}
		if err := withImportSavepoint(tx, func(tx *gorm.DB) error {
			return tx.Create(&usage).Error
		}); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Error importing DL usage row %d: %v", i+1, err))
			continue
		}
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

		var character Character
		if err := tx.Where("name = ?", charName).First(&character).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				result.Errors = append(result.Errors, fmt.Sprintf("Character not found: %s", charName))
			} else {
				result.Errors = append(result.Errors, fmt.Sprintf("Error importing transaction row %d: %v", i+1, err))
			}
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

		txEntry := Transaction{
			Date:        *dt,
			CharacterID: character.ID,
			Amount:      parseFloat(row[2]),
			Notes:       strings.TrimSpace(row[3]),
		}

		var existing Transaction
		err := tx.Where("date = ? AND character_id = ? AND amount = ? AND notes = ?",
			txEntry.Date, txEntry.CharacterID, txEntry.Amount, txEntry.Notes).First(&existing).Error
		if err == nil {
			result.TransactionsSkipped++
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			result.Errors = append(result.Errors, fmt.Sprintf("Error importing transaction row %d: %v", i+1, err))
			continue
		}
		if err := withImportSavepoint(tx, func(tx *gorm.DB) error {
			return tx.Create(&txEntry).Error
		}); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Error importing transaction row %d: %v", i+1, err))
			continue
		}
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

		var character Character
		if err := tx.Where("name = ?", charName).First(&character).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				result.Errors = append(result.Errors, fmt.Sprintf("Error importing cost of living row %d: %v", i+1, err))
			}
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
			if err := withImportSavepoint(tx, func(tx *gorm.DB) error {
				return tx.Model(&character).Updates(updates).Error
			}); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Error updating character %s: %v", charName, err))
			}
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

			cost := CostOfLiving{
				Date:        *dt,
				CharacterID: character.ID,
				Amount:      amount,
			}

			var existing CostOfLiving
			err := tx.Where("date = ? AND character_id = ? AND amount = ?",
				cost.Date, cost.CharacterID, cost.Amount).First(&existing).Error
			if err == nil {
				result.CostOfLivingsSkipped++
				continue
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				result.Errors = append(result.Errors, fmt.Sprintf("Error importing cost of living row %d: %v", i+1, err))
				continue
			}
			if err := withImportSavepoint(tx, func(tx *gorm.DB) error {
				return tx.Create(&cost).Error
			}); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Error importing cost of living row %d: %v", i+1, err))
				continue
			}
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

		var character Character
		if err := tx.Where("name = ?", charName).First(&character).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				result.Errors = append(result.Errors, fmt.Sprintf("Character not found: %s", charName))
			} else {
				result.Errors = append(result.Errors, fmt.Sprintf("Error importing registry row %d: %v", i+1, err))
			}
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

		registry := CharacterRegistry{
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

		var existing CharacterRegistry
		err := tx.Where("date = ? AND character_id = ? AND event = ?",
			registry.Date, registry.CharacterID, registry.Event).First(&existing).Error
		if err == nil {
			result.RegistriesSkipped++
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			result.Errors = append(result.Errors, fmt.Sprintf("Error importing registry row %d: %v", i+1, err))
			continue
		}
		if err := withImportSavepoint(tx, func(tx *gorm.DB) error {
			return tx.Create(&registry).Error
		}); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Error importing registry row %d: %v", i+1, err))
			continue
		}
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

	var currentMission *Mission

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
				mission := Mission{
					Date: *dt,
					DM:   dm,
					Name: eventName,
				}

				var existing Mission
				err := tx.Where("date = ? AND dm = ? AND name = ?",
					mission.Date, mission.DM, mission.Name).First(&existing).Error
				if err == nil {
					currentMission = &existing
					result.MissionsSkipped++
				} else if !errors.Is(err, gorm.ErrRecordNotFound) {
					result.Errors = append(result.Errors, fmt.Sprintf("Error importing mission %s: %v", mission.Name, err))
				} else if err := withImportSavepoint(tx, func(tx *gorm.DB) error {
					return tx.Create(&mission).Error
				}); err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("Error importing mission %s: %v", mission.Name, err))
				} else {
					currentMission = &mission
					result.Missions++
				}
			}
		}

		if currentMission == nil {
			continue
		}

		charName := strings.TrimSpace(row[2])
		if charName == "" {
			continue
		}

		var character Character
		if err := tx.Where("name = ?", charName).First(&character).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				result.Errors = append(result.Errors, fmt.Sprintf("Error importing mission entry row %d: %v", i+1, err))
			}
			continue
		}

		entry := MissionEntry{
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

		var existing MissionEntry
		err := tx.Where("mission_id = ? AND character_id = ?",
			entry.MissionID, entry.CharacterID).First(&existing).Error
		if err == nil {
			result.MissionEntriesSkipped++
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			result.Errors = append(result.Errors, fmt.Sprintf("Error importing mission entry row %d: %v", i+1, err))
			continue
		}
		if err := withImportSavepoint(tx, func(tx *gorm.DB) error {
			return tx.Create(&entry).Error
		}); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Error importing mission entry row %d: %v", i+1, err))
			continue
		}
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

	guildMap := make(map[string]*Guild)
	lastGuildName := ""

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
			lastGuildName = guildName
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

			guild := &Guild{
				Name:         guildName,
				CostOfLiving: col,
				Notes:        notes,
			}

			regDate := parseExcelDate(strings.TrimSpace(row[0]))
			if regDate != nil {
				guild.RegisteredAt = regDate
			}

			appDate := parseExcelDate(strings.TrimSpace(row[1]))
			if appDate != nil {
				guild.ApprovedAt = appDate
			}

			leaderName := strings.TrimSpace(row[3])
			skipGuildWrite := false
			if leaderName != "" {
				var leader Character
				if err := tx.Where("name = ?", leaderName).First(&leader).Error; err == nil {
					guild.LeaderID = &leader.ID
				} else if !errors.Is(err, gorm.ErrRecordNotFound) {
					result.Errors = append(result.Errors, fmt.Sprintf("Error importing guild %s: %v", guildName, err))
					skipGuildWrite = true
				}
			}

			if !skipGuildWrite {
				var existing Guild
				err := tx.Where("name = ?", guild.Name).First(&existing).Error
				if err == nil {
					guildMap[guildName] = &existing
					result.GuildsSkipped++
				} else if !errors.Is(err, gorm.ErrRecordNotFound) {
					result.Errors = append(result.Errors, fmt.Sprintf("Error importing guild %s: %v", guildName, err))
				} else if err := withImportSavepoint(tx, func(tx *gorm.DB) error {
					return tx.Create(guild).Error
				}); err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("Error importing guild %s: %v", guildName, err))
				} else {
					guildMap[guildName] = guild
					result.Guilds++
				}
			}

			if treasury != 0 {
				targetGuild := guildMap[guildName]
				if targetGuild != nil && targetGuild.ID != 0 {
					seedDate := regDate
					if seedDate == nil {
						seedDate = appDate
					}
					if seedDate == nil {
						now := time.Now()
						seedDate = &now
					}
					var created bool
					err := withImportSavepoint(tx, func(tx *gorm.DB) error {
						var cerr error
						created, cerr = CreateGuildTransaction(tx, GuildTransaction{
							Date:    *seedDate,
							GuildID: targetGuild.ID,
							Amount:  treasury,
							Notes:   "Registro Inicial",
						})
						return cerr
					})
					if err != nil {
						result.Errors = append(result.Errors, fmt.Sprintf("Error seeding treasury for %s: %v", guildName, err))
					} else if created {
						result.GuildTransactions++
					} else {
						result.GuildTransactionsSkipped++
					}
				}
			}
		}

		if memberName != "" {
			var member Character
			if err := tx.Where("name = ?", memberName).First(&member).Error; err != nil {
				if !errors.Is(err, gorm.ErrRecordNotFound) {
					result.Errors = append(result.Errors, fmt.Sprintf("Error importing guild member %s: %v", memberName, err))
				}
				continue
			}
			effectiveGuildName := guildName
			if effectiveGuildName == "" {
				effectiveGuildName = lastGuildName
			}
			if effectiveGuildName == "" {
				continue
			}
			target, ok := guildMap[effectiveGuildName]
			if !ok || target == nil {
				continue
			}
			// Idempotency: already member?
			var existing int64
			if err := tx.Raw("SELECT COUNT(*) FROM guild_members WHERE guild_id = ? AND character_id = ?", target.ID, member.ID).Scan(&existing).Error; err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Error importing guild member %s: %v", member.Name, err))
				continue
			}
			if existing > 0 {
				// Keep cache in sync even if already member
				if err := withImportSavepoint(tx, func(tx *gorm.DB) error {
					return SyncCharacterGuildName(tx, member.ID)
				}); err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("Error syncing guild name for %s: %v", member.Name, err))
				}
				continue
			}
			// Validate business rules: 15 per guild, 3 activos per player same guild, no cross-guild (activos only)
			if err := ValidateGuildJoin(tx, member.Player, target.ID, member.ID, member.Status); err != nil {
				if errors.Is(err, ErrGuildFull) {
					result.Errors = append(result.Errors, fmt.Sprintf("Gremio lleno (%s) - no se añadió %s: %v", target.Name, member.Name, err))
				} else if errors.Is(err, ErrPlayerLimit) {
					result.Errors = append(result.Errors, fmt.Sprintf("Jugador límite 3 activos (%s) - no se añadió %s a %s: %v", member.Player, member.Name, target.Name, err))
				} else if errors.Is(err, ErrCrossGuild) {
					result.Errors = append(result.Errors, fmt.Sprintf("Jugador en otro gremio (%s) - no se añadió %s a %s: %v", member.Player, member.Name, target.Name, err))
				} else {
					result.Errors = append(result.Errors, fmt.Sprintf("No se añadió %s a %s: %v", member.Name, target.Name, err))
				}
				continue
			}
			if err := withImportSavepoint(tx, func(tx *gorm.DB) error {
				if err := tx.Exec("INSERT INTO guild_members (guild_id, character_id) VALUES (?, ?)", target.ID, member.ID).Error; err != nil {
					return err
				}
				return SyncCharacterGuildName(tx, member.ID)
			}); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("No se añadió %s a %s: %v", member.Name, target.Name, err))
			}
		}
	}
}

func importGuildEconomy(tx *gorm.DB, f *excelize.File, result *ImportResult) {
	sheet := "Economía de Gremios"
	rows, err := f.GetRows(sheet)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Error reading %s: %v", sheet, err))
		return
	}

	for i, row := range rows {
		if i == 0 || len(row) < 3 {
			continue
		}

		guildName := strings.TrimSpace(row[1])
		if guildName == "" {
			continue
		}

		var guild Guild
		if err := tx.Where("name = ?", guildName).First(&guild).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				result.Errors = append(result.Errors, fmt.Sprintf("Guild not found: %s", guildName))
			} else {
				result.Errors = append(result.Errors, fmt.Sprintf("Error importing guild economy row %d: %v", i+1, err))
			}
			continue
		}

		dateStr := strings.TrimSpace(row[0])
		if dateStr == "" {
			continue
		}

		dt := parseExcelDate(dateStr)
		if dt == nil {
			continue
		}

		amount := parseFloat(row[2])
		if amount == 0 {
			continue
		}

		notes := ""
		if len(row) > 3 {
			notes = strings.TrimSpace(row[3])
		}

		var created bool
		err := withImportSavepoint(tx, func(tx *gorm.DB) error {
			var cerr error
			created, cerr = CreateGuildTransaction(tx, GuildTransaction{
				Date:    *dt,
				GuildID: guild.ID,
				Amount:  amount,
				Notes:   notes,
			})
			return cerr
		})
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Error importing guild economy row %d: %v", i+1, err))
			continue
		}
		if created {
			result.GuildTransactions++
		} else {
			result.GuildTransactionsSkipped++
		}
	}
}
