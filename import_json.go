package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	formatJSON = "json"
	formatXLSX = "xlsx"
)

var restoreWipeOrder = []string{
	"guild_members",
	"guild_transactions",
	"guilds",
	"mission_entries",
	"missions",
	"character_registries",
	"cost_of_livings",
	"transactions",
	"dl_usages",
	"characters",
}

type ExportMeta struct {
	Version    string
	ExportedAt time.Time
	TotalRows  int
}

func BuildExportMeta(data *ExportData) ExportMeta {
	return ExportMeta{
		Version:    data.Version,
		ExportedAt: data.ExportedAt,
		TotalRows: len(data.Characters) +
			len(data.DLUsages) +
			len(data.Transactions) +
			len(data.CostOfLivings) +
			len(data.Registries) +
			len(data.Missions) +
			len(data.MissionEntries) +
			len(data.Guilds) +
			len(data.GuildTransactions) +
			len(data.GuildMembers),
	}
}

func stripUTF8BOM(b []byte) []byte {
	return bytes.TrimPrefix(b, []byte{0xEF, 0xBB, 0xBF})
}

func detectImportFormat(filename string, head []byte) string {
	trimmed := bytes.TrimLeft(stripUTF8BOM(head), " \t\r\n")
	if len(trimmed) > 0 {
		if trimmed[0] == '{' || trimmed[0] == '[' {
			return formatJSON
		}
		if len(trimmed) >= 4 && trimmed[0] == 'P' && trimmed[1] == 'K' && trimmed[2] == 0x03 && trimmed[3] == 0x04 {
			return formatXLSX
		}
	}
	name := strings.ToLower(filename)
	if strings.HasSuffix(name, ".json") {
		return formatJSON
	}
	if strings.HasSuffix(name, ".xlsx") {
		return formatXLSX
	}
	return ""
}

func ValidateExportJSON(raw []byte) (*ExportData, error) {
	raw = stripUTF8BOM(raw)

	probe := map[string]json.RawMessage{}
	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil, fmt.Errorf("el archivo no es un JSON válido: %w", err)
	}
	if _, ok := probe["characters"]; !ok {
		if _, ok := probe["exported_at"]; !ok {
			return nil, fmt.Errorf("el archivo no parece un export de Archivist")
		}
	}

	data := &ExportData{}
	if err := json.Unmarshal(raw, data); err != nil {
		return nil, fmt.Errorf("no se pudo leer el export: %w", err)
	}
	return data, nil
}

func ImportJSON(data *ExportData) ImportResult {
	result := ImportResult{}
	if data == nil {
		result.Errors = append(result.Errors, "No hay datos para importar")
		return result
	}

	err := DB.Transaction(func(tx *gorm.DB) error {
		for _, table := range restoreWipeOrder {
			if err := tx.Exec("DELETE FROM " + table).Error; err != nil {
				return fmt.Errorf("vaciar %s: %w", table, err)
			}
		}

		for _, c := range data.Characters {
			row := c
			if err := tx.Create(&row).Error; err != nil {
				return fmt.Errorf("personaje %q: %w", c.Name, err)
			}
		}
		for _, e := range data.DLUsages {
			row := toDLUsage(e)
			if err := tx.Create(&row).Error; err != nil {
				return fmt.Errorf("uso de DL %d: %w", e.ID, err)
			}
		}
		for _, e := range data.Transactions {
			row := toTransaction(e)
			if err := tx.Create(&row).Error; err != nil {
				return fmt.Errorf("transacción %d: %w", e.ID, err)
			}
		}
		for _, e := range data.CostOfLivings {
			row := toCostOfLiving(e)
			if err := tx.Create(&row).Error; err != nil {
				return fmt.Errorf("costo de vida %d: %w", e.ID, err)
			}
		}
		for _, e := range data.Registries {
			row := toCharacterRegistry(e)
			if err := tx.Create(&row).Error; err != nil {
				return fmt.Errorf("registro de personaje %d: %w", e.ID, err)
			}
		}
		for _, e := range data.Missions {
			row := toMission(e)
			if err := tx.Create(&row).Error; err != nil {
				return fmt.Errorf("misión %d: %w", e.ID, err)
			}
		}
		for _, e := range data.MissionEntries {
			row := toMissionEntry(e)
			if err := tx.Create(&row).Error; err != nil {
				return fmt.Errorf("entrada de misión %d: %w", e.ID, err)
			}
		}
		for _, e := range data.Guilds {
			row := toGuild(e)
			if err := tx.Create(&row).Error; err != nil {
				return fmt.Errorf("gremio %q: %w", e.Name, err)
			}
		}
		for _, e := range data.GuildTransactions {
			row := toGuildTransaction(e)
			if err := tx.Create(&row).Error; err != nil {
				return fmt.Errorf("economía de gremio %d: %w", e.ID, err)
			}
		}
		for _, m := range data.GuildMembers {
			if err := tx.Exec("INSERT INTO guild_members (guild_id, character_id) VALUES (?, ?)", m.GuildID, m.CharacterID).Error; err != nil {
				return fmt.Errorf("miembro de gremio %d/%d: %w", m.GuildID, m.CharacterID, err)
			}
		}

		if err := RefreshCharacterStatsView(tx); err != nil {
			return fmt.Errorf("refrescar vista de estadísticas: %w", err)
		}
		return nil
	})
	if err != nil {
		result.Errors = append(result.Errors, "La restauración falló y se revirtió: "+err.Error())
		return result
	}

	result.Characters = len(data.Characters)
	result.DLUsages = len(data.DLUsages)
	result.Transactions = len(data.Transactions)
	result.CostOfLivings = len(data.CostOfLivings)
	result.Registries = len(data.Registries)
	result.Missions = len(data.Missions)
	result.MissionEntries = len(data.MissionEntries)
	result.Guilds = len(data.Guilds)
	result.GuildTransactions = len(data.GuildTransactions)
	result.GuildMembers = len(data.GuildMembers)
	return result
}

func toDLUsage(e ExportDLUsage) DLUsage {
	return DLUsage{
		ID:          e.ID,
		Date:        e.Date,
		CharacterID: e.CharacterID,
		DLUsed:      e.DLUsed,
		GoldChange:  e.GoldChange,
		Description: e.Description,
		CreatedAt:   e.CreatedAt,
	}
}

func toTransaction(e ExportTransaction) Transaction {
	return Transaction{
		ID:          e.ID,
		Date:        e.Date,
		CharacterID: e.CharacterID,
		Amount:      e.Amount,
		Notes:       e.Notes,
		CreatedAt:   e.CreatedAt,
	}
}

func toCostOfLiving(e ExportCostOfLiving) CostOfLiving {
	return CostOfLiving{
		ID:          e.ID,
		Date:        e.Date,
		CharacterID: e.CharacterID,
		Amount:      e.Amount,
		Notes:       e.Notes,
		CreatedAt:   e.CreatedAt,
	}
}

func toCharacterRegistry(e ExportCharacterRegistry) CharacterRegistry {
	return CharacterRegistry{
		ID:          e.ID,
		Date:        e.Date,
		CharacterID: e.CharacterID,
		Event:       e.Event,
		Experience:  e.Experience,
		Gold:        e.Gold,
		Renown:      e.Renown,
		Notes:       e.Notes,
		CreatedAt:   e.CreatedAt,
	}
}

func toMission(e ExportMission) Mission {
	return Mission{
		ID:        e.ID,
		Date:      e.Date,
		DM:        e.DM,
		Name:      e.Name,
		DeletedAt: e.DeletedAt,
		CreatedAt: e.CreatedAt,
	}
}

func toMissionEntry(e ExportMissionEntry) MissionEntry {
	return MissionEntry{
		ID:          e.ID,
		MissionID:   e.MissionID,
		CharacterID: e.CharacterID,
		XPMission:   e.XPMission,
		XPReport:    e.XPReport,
		XPGuild:     e.XPGuild,
		Gold:        e.Gold,
		Renown:      e.Renown,
		Notes:       e.Notes,
		DeletedAt:   e.DeletedAt,
		CreatedAt:   e.CreatedAt,
	}
}

func toGuild(e ExportGuild) Guild {
	return Guild{
		ID:           e.ID,
		Name:         e.Name,
		LeaderID:     e.LeaderID,
		HallType:     e.HallType,
		Notes:        e.Notes,
		CostOfLiving: e.CostOfLiving,
		Treasury:     e.Treasury,
		RegisteredAt: e.RegisteredAt,
		ApprovedAt:   e.ApprovedAt,
		CreatedAt:    e.CreatedAt,
		UpdatedAt:    e.UpdatedAt,
	}
}

func toGuildTransaction(e ExportGuildTransaction) GuildTransaction {
	return GuildTransaction{
		ID:        e.ID,
		Date:      e.Date,
		GuildID:   e.GuildID,
		Amount:    e.Amount,
		Notes:     e.Notes,
		CreatedAt: e.CreatedAt,
	}
}
