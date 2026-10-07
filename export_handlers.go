package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type ExportCounts struct {
	Characters        int64
	DLUsages          int64
	Transactions      int64
	CostOfLivings     int64
	Registries        int64
	Missions          int64
	MissionEntries    int64
	Guilds            int64
	GuildTransactions int64
	GuildMembers      int64
	CharacterStats    int64
}

func fetchExportCounts() (ExportCounts, error) {
	var counts ExportCounts
	tables := []struct {
		name string
		dest *int64
	}{
		{"characters", &counts.Characters},
		{"dl_usages", &counts.DLUsages},
		{"transactions", &counts.Transactions},
		{"cost_of_livings", &counts.CostOfLivings},
		{"character_registries", &counts.Registries},
		{"missions", &counts.Missions},
		{"mission_entries", &counts.MissionEntries},
		{"guilds", &counts.Guilds},
		{"guild_transactions", &counts.GuildTransactions},
		{"guild_members", &counts.GuildMembers},
		{"character_stats", &counts.CharacterStats},
	}
	for _, t := range tables {
		if err := DB.Table(t.name).Count(t.dest).Error; err != nil {
			return counts, fmt.Errorf("count %s: %w", t.name, err)
		}
	}
	return counts, nil
}

func ExportPageHandler(c *gin.Context) {
	counts, err := fetchExportCounts()
	if err != nil {
		render(c, http.StatusInternalServerError, "export.html", gin.H{
			"Title":      "Exportar Datos",
			"ActiveMenu": "export",
			"Error":      "Error al contar registros: " + err.Error(),
		})
		return
	}
	render(c, http.StatusOK, "export.html", gin.H{
		"Title":      "Exportar Datos",
		"ActiveMenu": "export",
		"Counts":     counts,
	})
}

func ExportDownloadHandler(c *gin.Context) {
	data, err := ExportAll(DB)
	if err != nil {
		counts, _ := fetchExportCounts()
		render(c, http.StatusInternalServerError, "export.html", gin.H{
			"Title":      "Exportar Datos",
			"ActiveMenu": "export",
			"Counts":     counts,
			"Error":      "Error al exportar datos: " + err.Error(),
		})
		return
	}

	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		counts, _ := fetchExportCounts()
		render(c, http.StatusInternalServerError, "export.html", gin.H{
			"Title":      "Exportar Datos",
			"ActiveMenu": "export",
			"Counts":     counts,
			"Error":      "Error al serializar JSON: " + err.Error(),
		})
		return
	}

	filename := fmt.Sprintf("archivist-export-%s.json", time.Now().Format("20060102-150405"))
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.Data(http.StatusOK, "application/json", jsonBytes)
}
