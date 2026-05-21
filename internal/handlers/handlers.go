package handlers

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"codeberg.org/chewrafa/archivist/internal/db"
	"codeberg.org/chewrafa/archivist/internal/models"
	"codeberg.org/chewrafa/archivist/internal/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type PageData struct {
	Title string
	Data  interface{}
}

func IndexHandler(c *gin.Context) {
	stats, _ := services.GetAllCharactersWithStats()

	activeCount := 0
	levelDist := make(map[int]int)
	classDist := make(map[string]int)
	speciesDist := make(map[string]int)

	for _, s := range stats {
		if s.Status == "Activo" {
			activeCount++
		}
		levelDist[s.Level]++
		classDist[s.Class]++
		speciesDist[s.Species]++
	}

	var recentMissions []models.Mission
	db.DB.Order("date DESC").Limit(5).Preload("Entries").Find(&recentMissions)

	var recentDLUsages []models.DLUsage
	db.DB.Order("date DESC").Limit(5).Preload("Character").Find(&recentDLUsages)

	var recentTransactions []models.Transaction
	db.DB.Order("date DESC").Limit(5).Preload("Character").Find(&recentTransactions)

	render(c, http.StatusOK, "index.html", gin.H{
		"Title":              "Dashboard",
		"Stats":              stats,
		"ActiveCount":        activeCount,
		"TotalCount":         len(stats),
		"LevelDist":          levelDist,
		"ClassDist":          classDist,
		"SpeciesDist":        speciesDist,
		"RecentMissions":     recentMissions,
		"RecentDLUsages":     recentDLUsages,
		"RecentTransactions": recentTransactions,
	})
}

func CharactersHandler(c *gin.Context) {
	stats, _ := services.GetAllCharactersWithStats()

	filter := c.Query("filter")
	if filter != "" {
		var filtered []services.CharacterStats
		for _, s := range stats {
			if strings.EqualFold(s.Status, filter) || strings.EqualFold(s.Class, filter) || strings.EqualFold(s.Species, filter) {
				filtered = append(filtered, s)
			}
		}
		stats = filtered
	}

	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Name < stats[j].Name
	})

	render(c, http.StatusOK, "characters.html", gin.H{
		"Title":  "Personajes",
		"Stats":  stats,
		"Filter": filter,
	})
}

func CharacterDetailHandler(c *gin.Context) {
	id := c.Param("id")
	stats, err := services.GetCharacterWithStats(parseUint(id))
	if err != nil {
		c.Redirect(http.StatusFound, "/characters")
		return
	}

	var registries []models.CharacterRegistry
	db.DB.Where("character_id = ?", id).Order("date DESC").Find(&registries)

	var missionEntries []models.MissionEntry
	db.DB.Where("character_id = ?", id).Order("created_at DESC").Preload("Mission").Find(&missionEntries)

	var dlUsages []models.DLUsage
	db.DB.Where("character_id = ?", id).Order("date DESC").Find(&dlUsages)

	var transactions []models.Transaction
	db.DB.Where("character_id = ?", id).Order("date DESC").Find(&transactions)

	render(c, http.StatusOK, "character-detail.html", gin.H{
		"Title":          stats.Name,
		"Stats":          stats,
		"Registries":     registries,
		"MissionEntries": missionEntries,
		"DLUsages":       dlUsages,
		"Transactions":   transactions,
	})
}

func MissionsHandler(c *gin.Context) {
	var missions []models.Mission
	db.DB.Order("date DESC").Preload("Entries").Preload("Entries.Character").Find(&missions)

	render(c, http.StatusOK, "missions.html", gin.H{
		"Title":    "Misiones",
		"Missions": missions,
	})
}

func DLHandler(c *gin.Context) {
	var usages []models.DLUsage
	db.DB.Order("date DESC").Preload("Character").Limit(50).Find(&usages)

	var characters []models.Character
	db.DB.Order("name ASC").Find(&characters)

	render(c, http.StatusOK, "dl.html", gin.H{
		"Title":      "Uso de DL",
		"Usages":     usages,
		"Characters": characters,
	})
}

type DLUsageFormData struct {
	Date        string
	CharacterID uint
	DLUsed      int
	GoldChange  float64
	Description string
}

type TransactionFormData struct {
	Date        string
	CharacterID uint
	Amount      float64
	Notes       string
}

func DLUsageCreateHandler(c *gin.Context) {
	form := DLUsageFormData{
		Date:        c.PostForm("date"),
		CharacterID: parseUint(c.PostForm("character_id")),
		DLUsed:      parseInt(c.PostForm("dl_used")),
		GoldChange:  parseFloat(c.PostForm("gold_change")),
		Description: c.PostForm("description"),
	}

	if form.Date == "" || form.CharacterID == 0 || form.DLUsed <= 0 {
		c.Redirect(http.StatusFound, "/dl")
		return
	}

	date, err := time.Parse("2006-01-02", form.Date)
	if err != nil {
		c.Redirect(http.StatusFound, "/dl")
		return
	}

	usage := models.DLUsage{
		Date:        date,
		CharacterID: form.CharacterID,
		DLUsed:      form.DLUsed,
		GoldChange:  form.GoldChange,
		Description: form.Description,
	}

	db.DB.Create(&usage)
	c.Redirect(http.StatusFound, "/dl")
}

func DLUsageEditHandler(c *gin.Context) {
	id := c.Param("id")
	var usage models.DLUsage
	if err := db.DB.First(&usage, id).Error; err != nil {
		c.Redirect(http.StatusFound, "/dl")
		return
	}

	var characters []models.Character
	db.DB.Order("name ASC").Find(&characters)

	render(c, http.StatusOK, "dl-usage-form.html", gin.H{
		"Title":       "Editar Uso de DL",
		"ActiveMenu":  "dl",
		"Action":      "/dl/usages/" + id,
		"SubmitLabel": "Actualizar",
		"Form": DLUsageFormData{
			Date:        usage.Date.Format("2006-01-02"),
			CharacterID: usage.CharacterID,
			DLUsed:      usage.DLUsed,
			GoldChange:  usage.GoldChange,
			Description: usage.Description,
		},
		"Characters": characters,
		"Error":      "",
	})
}

func DLUsageUpdateHandler(c *gin.Context) {
	id := c.Param("id")
	form := DLUsageFormData{
		Date:        c.PostForm("date"),
		CharacterID: parseUint(c.PostForm("character_id")),
		DLUsed:      parseInt(c.PostForm("dl_used")),
		GoldChange:  parseFloat(c.PostForm("gold_change")),
		Description: c.PostForm("description"),
	}

	if form.Date == "" || form.CharacterID == 0 || form.DLUsed <= 0 {
		var characters []models.Character
		db.DB.Order("name ASC").Find(&characters)
		render(c, http.StatusBadRequest, "dl-usage-form.html", gin.H{
			"Title":       "Editar Uso de DL",
		"ActiveMenu":  "dl",
			"Action":      "/dl/usages/" + id,
			"SubmitLabel": "Actualizar",
			"Form":        form,
			"Characters":  characters,
			"Error":       "Fecha, personaje y DL son obligatorios",
		})
		return
	}

	date, err := time.Parse("2006-01-02", form.Date)
	if err != nil {
		var characters []models.Character
		db.DB.Order("name ASC").Find(&characters)
		render(c, http.StatusBadRequest, "dl-usage-form.html", gin.H{
			"Title":       "Editar Uso de DL",
		"ActiveMenu":  "dl",
			"Action":      "/dl/usages/" + id,
			"SubmitLabel": "Actualizar",
			"Form":        form,
			"Characters":  characters,
			"Error":       "Fecha inválida",
		})
		return
	}

	var usage models.DLUsage
	if err := db.DB.First(&usage, id).Error; err != nil {
		c.Redirect(http.StatusFound, "/dl")
		return
	}

	usage.Date = date
	usage.CharacterID = form.CharacterID
	usage.DLUsed = form.DLUsed
	usage.GoldChange = form.GoldChange
	usage.Description = form.Description

	if err := db.DB.Save(&usage).Error; err != nil {
		var characters []models.Character
		db.DB.Order("name ASC").Find(&characters)
		render(c, http.StatusInternalServerError, "dl-usage-form.html", gin.H{
			"Title":       "Editar Uso de DL",
		"ActiveMenu":  "dl",
			"Action":      "/dl/usages/" + id,
			"SubmitLabel": "Actualizar",
			"Form":        form,
			"Characters":  characters,
			"Error":       "Error al actualizar",
		})
		return
	}

	c.Redirect(http.StatusFound, "/dl")
}

func DLUsageDeleteHandler(c *gin.Context) {
	id := c.Param("id")
	db.DB.Delete(&models.DLUsage{}, id)
	c.Redirect(http.StatusFound, "/dl")
}

func TransactionsHandler(c *gin.Context) {
	var transactions []models.Transaction
	db.DB.Order("date DESC").Preload("Character").Limit(100).Find(&transactions)

	var characters []models.Character
	db.DB.Order("name ASC").Find(&characters)

	render(c, http.StatusOK, "transactions.html", gin.H{
		"Title":        "Transacciones",
		"Transactions": transactions,
		"Characters":   characters,
	})
}

func TransactionCreateHandler(c *gin.Context) {
	date := c.PostForm("date")
	characterID := parseUint(c.PostForm("character_id"))
	amount := parseFloat(c.PostForm("amount"))
	notes := c.PostForm("notes")

	if date == "" || characterID == 0 {
		c.Redirect(http.StatusFound, "/transactions")
		return
	}

	parsed, err := time.Parse("2006-01-02", date)
	if err != nil {
		c.Redirect(http.StatusFound, "/transactions")
		return
	}

	tx := models.Transaction{
		Date:        parsed,
		CharacterID: characterID,
		Amount:      amount,
		Notes:       notes,
	}

	db.DB.Create(&tx)
	c.Redirect(http.StatusFound, "/transactions")
}

func TransactionEditHandler(c *gin.Context) {
	id := c.Param("id")
	var tx models.Transaction
	if err := db.DB.First(&tx, id).Error; err != nil {
		c.Redirect(http.StatusFound, "/transactions")
		return
	}

	var characters []models.Character
	db.DB.Order("name ASC").Find(&characters)

	render(c, http.StatusOK, "transaction-form.html", gin.H{
		"Title":       "Editar Transacción",
		"ActiveMenu":  "transactions",
		"Action":      "/transactions/detail/" + id,
		"SubmitLabel": "Actualizar Transacción",
		"Form": TransactionFormData{
			Date:        tx.Date.Format("2006-01-02"),
			CharacterID: tx.CharacterID,
			Amount:      tx.Amount,
			Notes:       tx.Notes,
		},
		"Characters": characters,
		"Error":      "",
	})
}

func TransactionUpdateHandler(c *gin.Context) {
	id := c.Param("id")
	form := TransactionFormData{
		Date:        c.PostForm("date"),
		CharacterID: parseUint(c.PostForm("character_id")),
		Amount:      parseFloat(c.PostForm("amount")),
		Notes:       c.PostForm("notes"),
	}

	if form.Date == "" || form.CharacterID == 0 {
		var characters []models.Character
		db.DB.Order("name ASC").Find(&characters)
		render(c, http.StatusBadRequest, "transaction-form.html", gin.H{
			"Title":       "Editar Transacción",
		"ActiveMenu":  "transactions",
			"Action":      "/transactions/detail/" + id,
			"SubmitLabel": "Actualizar Transacción",
			"Form":        form,
			"Characters":  characters,
			"Error":       "Fecha y personaje son obligatorios",
		})
		return
	}

	date, err := time.Parse("2006-01-02", form.Date)
	if err != nil {
		var characters []models.Character
		db.DB.Order("name ASC").Find(&characters)
		render(c, http.StatusBadRequest, "transaction-form.html", gin.H{
			"Title":       "Editar Transacción",
		"ActiveMenu":  "transactions",
			"Action":      "/transactions/detail/" + id,
			"SubmitLabel": "Actualizar Transacción",
			"Form":        form,
			"Characters":  characters,
			"Error":       "Fecha inválida",
		})
		return
	}

	var tx models.Transaction
	if err := db.DB.First(&tx, id).Error; err != nil {
		c.Redirect(http.StatusFound, "/transactions")
		return
	}

	tx.Date = date
	tx.CharacterID = form.CharacterID
	tx.Amount = form.Amount
	tx.Notes = form.Notes

	if err := db.DB.Save(&tx).Error; err != nil {
		var characters []models.Character
		db.DB.Order("name ASC").Find(&characters)
		render(c, http.StatusInternalServerError, "transaction-form.html", gin.H{
			"Title":       "Editar Transacción",
		"ActiveMenu":  "transactions",
			"Action":      "/transactions/detail/" + id,
			"SubmitLabel": "Actualizar Transacción",
			"Form":        form,
			"Characters":  characters,
			"Error":       "Error al actualizar la transacción",
		})
		return
	}

	c.Redirect(http.StatusFound, "/transactions")
}

func TransactionDeleteHandler(c *gin.Context) {
	id := c.Param("id")
	db.DB.Delete(&models.Transaction{}, id)
	c.Redirect(http.StatusFound, "/transactions")
}

func GuildsHandler(c *gin.Context) {
	var guilds []models.Guild
	db.DB.Preload("Leader").Preload("Members").Find(&guilds)

	render(c, http.StatusOK, "guilds.html", gin.H{
		"Title":  "Gremios",
		"Guilds": guilds,
	})
}

type GuildFormData struct {
	Name         string
	LeaderID     uint
	HallType     string
	Notes        string
	CostOfLiving float64
	Treasury     float64
	RegisteredAt string
	ApprovedAt   string
}

func GuildNewHandler(c *gin.Context) {
	var characters []models.Character
	db.DB.Order("name ASC").Find(&characters)

	render(c, http.StatusOK, "guild-form.html", gin.H{
		"Title":       "Nuevo Gremio",
		"ActiveMenu":  "guilds",
		"Action":      "/guilds",
		"SubmitLabel": "Crear Gremio",
		"Form":        GuildFormData{},
		"Characters":  characters,
		"Error":       "",
	})
}

func GuildCreateHandler(c *gin.Context) {
	form := GuildFormData{
		Name:         c.PostForm("name"),
		LeaderID:     parseUint(c.PostForm("leader_id")),
		HallType:     c.PostForm("hall_type"),
		Notes:        c.PostForm("notes"),
		CostOfLiving: parseFloat(c.PostForm("cost_of_living")),
		Treasury:     parseFloat(c.PostForm("treasury")),
		RegisteredAt: c.PostForm("registered_at"),
		ApprovedAt:   c.PostForm("approved_at"),
	}

	if form.Name == "" {
		var characters []models.Character
		db.DB.Order("name ASC").Find(&characters)
		render(c, http.StatusBadRequest, "guild-form.html", gin.H{
			"Title":       "Nuevo Gremio",
		"ActiveMenu":  "guilds",
			"Action":      "/guilds",
			"SubmitLabel": "Crear Gremio",
			"Form":        form,
			"Characters":  characters,
			"Error":       "El nombre es obligatorio",
		})
		return
	}

	guild := models.Guild{
		Name:        form.Name,
		HallType:    form.HallType,
		Notes:       form.Notes,
		CostOfLiving: form.CostOfLiving,
		Treasury:    form.Treasury,
	}

	if form.LeaderID != 0 {
		guild.LeaderID = &form.LeaderID
	}

	if form.RegisteredAt != "" {
		t, err := time.Parse("2006-01-02", form.RegisteredAt)
		if err == nil {
			guild.RegisteredAt = &t
		}
	}

	if form.ApprovedAt != "" {
		t, err := time.Parse("2006-01-02", form.ApprovedAt)
		if err == nil {
			guild.ApprovedAt = &t
		}
	}

	if err := db.DB.Create(&guild).Error; err != nil {
		var characters []models.Character
		db.DB.Order("name ASC").Find(&characters)
		render(c, http.StatusInternalServerError, "guild-form.html", gin.H{
			"Title":       "Nuevo Gremio",
		"ActiveMenu":  "guilds",
			"Action":      "/guilds",
			"SubmitLabel": "Crear Gremio",
			"Form":        form,
			"Characters":  characters,
			"Error":       "Error al crear el gremio",
		})
		return
	}

	c.Redirect(http.StatusFound, "/guilds/detail/"+fmt.Sprint(guild.ID))
}

func GuildDetailHandler(c *gin.Context) {
	id := c.Param("id")
	var guild models.Guild
	if err := db.DB.Preload("Leader").Preload("Members").First(&guild, id).Error; err != nil {
		c.Redirect(http.StatusFound, "/guilds")
		return
	}

	render(c, http.StatusOK, "guild-detail.html", gin.H{
		"Title": guild.Name,
		"ActiveMenu":  "guilds",
		"Guild": guild,
	})
}

func GuildEditHandler(c *gin.Context) {
	id := c.Param("id")
	var guild models.Guild
	if err := db.DB.Preload("Leader").First(&guild, id).Error; err != nil {
		c.Redirect(http.StatusFound, "/guilds")
		return
	}

	var characters []models.Character
	db.DB.Order("name ASC").Find(&characters)

	form := GuildFormData{
		Name:         guild.Name,
		HallType:     guild.HallType,
		Notes:        guild.Notes,
		CostOfLiving: guild.CostOfLiving,
		Treasury:     guild.Treasury,
	}

	if guild.LeaderID != nil {
		form.LeaderID = *guild.LeaderID
	}

	if guild.RegisteredAt != nil {
		form.RegisteredAt = guild.RegisteredAt.Format("2006-01-02")
	}

	if guild.ApprovedAt != nil {
		form.ApprovedAt = guild.ApprovedAt.Format("2006-01-02")
	}

	render(c, http.StatusOK, "guild-form.html", gin.H{
		"Title":       "Editar Gremio",
		"ActiveMenu":  "guilds",
		"Action":      "/guilds/detail/" + id,
		"SubmitLabel": "Actualizar Gremio",
		"Form":        form,
		"Characters":  characters,
		"Error":       "",
	})
}

func GuildUpdateHandler(c *gin.Context) {
	id := c.Param("id")
	form := GuildFormData{
		Name:         c.PostForm("name"),
		LeaderID:     parseUint(c.PostForm("leader_id")),
		HallType:     c.PostForm("hall_type"),
		Notes:        c.PostForm("notes"),
		CostOfLiving: parseFloat(c.PostForm("cost_of_living")),
		Treasury:     parseFloat(c.PostForm("treasury")),
		RegisteredAt: c.PostForm("registered_at"),
		ApprovedAt:   c.PostForm("approved_at"),
	}

	if form.Name == "" {
		var characters []models.Character
		db.DB.Order("name ASC").Find(&characters)
		render(c, http.StatusBadRequest, "guild-form.html", gin.H{
			"Title":       "Editar Gremio",
		"ActiveMenu":  "guilds",
			"Action":      "/guilds/detail/" + id,
			"SubmitLabel": "Actualizar Gremio",
			"Form":        form,
			"Characters":  characters,
			"Error":       "El nombre es obligatorio",
		})
		return
	}

	var guild models.Guild
	if err := db.DB.First(&guild, id).Error; err != nil {
		c.Redirect(http.StatusFound, "/guilds")
		return
	}

	guild.Name = form.Name
	guild.HallType = form.HallType
	guild.Notes = form.Notes
	guild.CostOfLiving = form.CostOfLiving
	guild.Treasury = form.Treasury

	if form.LeaderID != 0 {
		guild.LeaderID = &form.LeaderID
	} else {
		guild.LeaderID = nil
	}

	if form.RegisteredAt != "" {
		t, err := time.Parse("2006-01-02", form.RegisteredAt)
		if err == nil {
			guild.RegisteredAt = &t
		}
	} else {
		guild.RegisteredAt = nil
	}

	if form.ApprovedAt != "" {
		t, err := time.Parse("2006-01-02", form.ApprovedAt)
		if err == nil {
			guild.ApprovedAt = &t
		}
	} else {
		guild.ApprovedAt = nil
	}

	if err := db.DB.Save(&guild).Error; err != nil {
		var characters []models.Character
		db.DB.Order("name ASC").Find(&characters)
		render(c, http.StatusInternalServerError, "guild-form.html", gin.H{
			"Title":       "Editar Gremio",
		"ActiveMenu":  "guilds",
			"Action":      "/guilds/detail/" + id,
			"SubmitLabel": "Actualizar Gremio",
			"Form":        form,
			"Characters":  characters,
			"Error":       "Error al actualizar el gremio",
		})
		return
	}

	c.Redirect(http.StatusFound, "/guilds/detail/"+id)
}

func GuildDeleteHandler(c *gin.Context) {
	id := c.Param("id")
	db.DB.Delete(&models.Guild{}, id)
	c.Redirect(http.StatusFound, "/guilds")
}

type MissionFormData struct {
	Date string
	DM   string
	Name string
}

type MissionEntryFormData struct {
	CharacterID uint
	XPMission   float64
	XPReport    float64
	XPGuild     float64
	Gold        float64
	Renown      float64
	Notes       string
}

func MissionNewHandler(c *gin.Context) {
	render(c, http.StatusOK, "mission-form.html", gin.H{
		"Title":       "Nueva Misión",
		"ActiveMenu":  "missions",
		"Action":      "/missions",
		"SubmitLabel": "Crear Misión",
		"Form":        MissionFormData{},
		"Error":       "",
	})
}

func MissionCreateHandler(c *gin.Context) {
	form := MissionFormData{
		Date: c.PostForm("date"),
		DM:   c.PostForm("dm"),
		Name: c.PostForm("name"),
	}

	if form.Name == "" || form.DM == "" {
		render(c, http.StatusBadRequest, "mission-form.html", gin.H{
			"Title":       "Nueva Misión",
		"ActiveMenu":  "missions",
			"Action":      "/missions",
			"SubmitLabel": "Crear Misión",
			"Form":        form,
			"Error":       "El nombre y el DM son obligatorios",
		})
		return
	}

	date, err := time.Parse("2006-01-02", form.Date)
	if err != nil {
		render(c, http.StatusBadRequest, "mission-form.html", gin.H{
			"Title":       "Nueva Misión",
		"ActiveMenu":  "missions",
			"Action":      "/missions",
			"SubmitLabel": "Crear Misión",
			"Form":        form,
			"Error":       "Fecha inválida",
		})
		return
	}

	mission := models.Mission{
		Date: date,
		DM:   form.DM,
		Name: form.Name,
	}

	if err := db.DB.Create(&mission).Error; err != nil {
		render(c, http.StatusInternalServerError, "mission-form.html", gin.H{
			"Title":       "Nueva Misión",
		"ActiveMenu":  "missions",
			"Action":      "/missions",
			"SubmitLabel": "Crear Misión",
			"Form":        form,
			"Error":       "Error al crear la misión",
		})
		return
	}

	c.Redirect(http.StatusFound, "/missions/detail/"+fmt.Sprint(mission.ID))
}

func MissionDetailHandler(c *gin.Context) {
	id := c.Param("id")
	var mission models.Mission
	if err := db.DB.Preload("Entries", func(db *gorm.DB) *gorm.DB {
		return db.Where("deleted_at IS NULL").Order("created_at ASC")
	}).Preload("Entries.Character").First(&mission, id).Error; err != nil {
		c.Redirect(http.StatusFound, "/missions")
		return
	}

	var characters []models.Character
	db.DB.Order("name ASC").Find(&characters)

	render(c, http.StatusOK, "mission-detail.html", gin.H{
		"Title":      mission.Name,
		"ActiveMenu":  "missions",
		"Mission":    mission,
		"Characters": characters,
		"EntryForm":  MissionEntryFormData{},
		"Error":      "",
	})
}

func MissionEditHandler(c *gin.Context) {
	id := c.Param("id")
	var mission models.Mission
	if err := db.DB.First(&mission, id).Error; err != nil {
		c.Redirect(http.StatusFound, "/missions")
		return
	}

	render(c, http.StatusOK, "mission-form.html", gin.H{
		"Title":       "Editar Misión",
		"ActiveMenu":  "missions",
		"Action":      "/missions/detail/" + id,
		"SubmitLabel": "Actualizar Misión",
		"Form": MissionFormData{
			Date: mission.Date.Format("2006-01-02"),
			DM:   mission.DM,
			Name: mission.Name,
		},
		"Error": "",
	})
}

func MissionUpdateHandler(c *gin.Context) {
	id := c.Param("id")
	form := MissionFormData{
		Date: c.PostForm("date"),
		DM:   c.PostForm("dm"),
		Name: c.PostForm("name"),
	}

	if form.Name == "" || form.DM == "" {
		render(c, http.StatusBadRequest, "mission-form.html", gin.H{
			"Title":       "Editar Misión",
		"ActiveMenu":  "missions",
			"Action":      "/missions/detail/" + id,
			"SubmitLabel": "Actualizar Misión",
			"Form":        form,
			"Error":       "El nombre y el DM son obligatorios",
		})
		return
	}

	date, err := time.Parse("2006-01-02", form.Date)
	if err != nil {
		render(c, http.StatusBadRequest, "mission-form.html", gin.H{
			"Title":       "Editar Misión",
		"ActiveMenu":  "missions",
			"Action":      "/missions/detail/" + id,
			"SubmitLabel": "Actualizar Misión",
			"Form":        form,
			"Error":       "Fecha inválida",
		})
		return
	}

	var mission models.Mission
	if err := db.DB.First(&mission, id).Error; err != nil {
		c.Redirect(http.StatusFound, "/missions")
		return
	}

	mission.Date = date
	mission.DM = form.DM
	mission.Name = form.Name

	if err := db.DB.Save(&mission).Error; err != nil {
		render(c, http.StatusInternalServerError, "mission-form.html", gin.H{
			"Title":       "Editar Misión",
		"ActiveMenu":  "missions",
			"Action":      "/missions/detail/" + id,
			"SubmitLabel": "Actualizar Misión",
			"Form":        form,
			"Error":       "Error al actualizar la misión",
		})
		return
	}

	c.Redirect(http.StatusFound, "/missions/detail/"+id)
}

func MissionDeleteHandler(c *gin.Context) {
	id := c.Param("id")

	var entries []models.MissionEntry
	db.DB.Where("mission_id = ?", id).Find(&entries)
	for _, entry := range entries {
		db.DB.Delete(&entry)
	}

	db.DB.Delete(&models.Mission{}, id)
	c.Redirect(http.StatusFound, "/missions")
}

func MissionEntryCreateHandler(c *gin.Context) {
	id := c.Param("id")

	var mission models.Mission
	if err := db.DB.First(&mission, id).Error; err != nil {
		c.Redirect(http.StatusFound, "/missions")
		return
	}

	entry := models.MissionEntry{
		MissionID:   mission.ID,
		CharacterID: parseUint(c.PostForm("character_id")),
		XPMission:   parseFloat(c.PostForm("xp_mission")),
		XPReport:    parseFloat(c.PostForm("xp_report")),
		XPGuild:     parseFloat(c.PostForm("xp_guild")),
		Gold:        parseFloat(c.PostForm("gold")),
		Renown:      parseFloat(c.PostForm("renown")),
		Notes:       c.PostForm("notes"),
	}

	db.DB.Create(&entry)
	c.Redirect(http.StatusFound, "/missions/detail/"+id)
}

func MissionEntryEditHandler(c *gin.Context) {
	id := c.Param("id")
	eid := c.Param("eid")

	var mission models.Mission
	if err := db.DB.First(&mission, id).Error; err != nil {
		c.Redirect(http.StatusFound, "/missions")
		return
	}

	var entry models.MissionEntry
	if err := db.DB.Preload("Character").First(&entry, eid).Error; err != nil {
		c.Redirect(http.StatusFound, "/missions/detail/"+id)
		return
	}

	var characters []models.Character
	db.DB.Order("name ASC").Find(&characters)

	render(c, http.StatusOK, "mission-entry-form.html", gin.H{
		"Title":      "Editar Entrada de Misión",
		"ActiveMenu":  "missions",
		"Mission":    mission,
		"Entry":      entry,
		"Characters": characters,
		"Error":      "",
	})
}

func MissionEntryUpdateHandler(c *gin.Context) {
	id := c.Param("id")
	eid := c.Param("eid")

	var entry models.MissionEntry
	if err := db.DB.First(&entry, eid).Error; err != nil {
		c.Redirect(http.StatusFound, "/missions/detail/"+id)
		return
	}

	entry.CharacterID = parseUint(c.PostForm("character_id"))
	entry.XPMission = parseFloat(c.PostForm("xp_mission"))
	entry.XPReport = parseFloat(c.PostForm("xp_report"))
	entry.XPGuild = parseFloat(c.PostForm("xp_guild"))
	entry.Gold = parseFloat(c.PostForm("gold"))
	entry.Renown = parseFloat(c.PostForm("renown"))
	entry.Notes = c.PostForm("notes")

	db.DB.Save(&entry)
	c.Redirect(http.StatusFound, "/missions/detail/"+id)
}

func MissionEntryDeleteHandler(c *gin.Context) {
	id := c.Param("id")
	eid := c.Param("eid")
	db.DB.Delete(&models.MissionEntry{}, eid)
	c.Redirect(http.StatusFound, "/missions/detail/"+id)
}

type FormData struct {
	Number    int
	Player    string
	Name      string
	Status    string
	Species   string
	Class     string
	GuildName string
}

func CharacterNewHandler(c *gin.Context) {
	render(c, http.StatusOK, "character-form.html", gin.H{
		"Title":       "Nuevo Personaje",
		"ActiveMenu":  "characters",
		"Action":      "/characters",
		"SubmitLabel": "Crear Personaje",
		"Form":        FormData{Status: "Activo"},
		"Error":       "",
	})
}

func CharacterCreateHandler(c *gin.Context) {
	form := FormData{
		Number:    parseInt(c.PostForm("number")),
		Player:    c.PostForm("player"),
		Name:      c.PostForm("name"),
		Status:    c.PostForm("status"),
		Species:   c.PostForm("species"),
		Class:     c.PostForm("class"),
		GuildName: c.PostForm("guild_name"),
	}

	if form.Name == "" {
		render(c, http.StatusBadRequest, "character-form.html", gin.H{
			"Title":       "Nuevo Personaje",
		"ActiveMenu":  "characters",
			"Action":      "/characters",
			"SubmitLabel": "Crear Personaje",
			"Form":        form,
			"Error":       "El nombre es obligatorio",
		})
		return
	}

	var existing models.Character
	if err := db.DB.Unscoped().Where("name = ? AND deleted_at IS NULL", form.Name).First(&existing).Error; err == nil {
		render(c, http.StatusConflict, "character-form.html", gin.H{
			"Title":       "Nuevo Personaje",
		"ActiveMenu":  "characters",
			"Action":      "/characters",
			"SubmitLabel": "Crear Personaje",
			"Form":        form,
			"Error":       "Ya existe un personaje con ese nombre",
		})
		return
	}

	character := models.Character{
		Number:    form.Number,
		Player:    form.Player,
		Name:      form.Name,
		Status:    form.Status,
		Species:   form.Species,
		Class:     form.Class,
		GuildName: form.GuildName,
	}

	if err := db.DB.Create(&character).Error; err != nil {
		render(c, http.StatusInternalServerError, "character-form.html", gin.H{
			"Title":       "Nuevo Personaje",
		"ActiveMenu":  "characters",
			"Action":      "/characters",
			"SubmitLabel": "Crear Personaje",
			"Form":        form,
			"Error":       "Error al crear el personaje",
		})
		return
	}

	c.Redirect(http.StatusFound, "/characters/detail/"+fmt.Sprint(character.ID))
}

func CharacterEditHandler(c *gin.Context) {
	id := c.Param("id")
	var character models.Character
	if err := db.DB.Unscoped().First(&character, id).Error; err != nil {
		c.Redirect(http.StatusFound, "/characters")
		return
	}

	render(c, http.StatusOK, "character-form.html", gin.H{
		"Title":       "Editar Personaje",
		"ActiveMenu":  "characters",
		"Action":      "/characters/detail/" + id,
		"SubmitLabel": "Actualizar Personaje",
		"Form": FormData{
			Number:    character.Number,
			Player:    character.Player,
			Name:      character.Name,
			Status:    character.Status,
			Species:   character.Species,
			Class:     character.Class,
			GuildName: character.GuildName,
		},
		"Error": "",
	})
}

func CharacterUpdateHandler(c *gin.Context) {
	id := c.Param("id")

	form := FormData{
		Number:    parseInt(c.PostForm("number")),
		Player:    c.PostForm("player"),
		Name:      c.PostForm("name"),
		Status:    c.PostForm("status"),
		Species:   c.PostForm("species"),
		Class:     c.PostForm("class"),
		GuildName: c.PostForm("guild_name"),
	}

	if form.Name == "" {
		render(c, http.StatusBadRequest, "character-form.html", gin.H{
			"Title":       "Editar Personaje",
		"ActiveMenu":  "characters",
			"Action":      "/characters/detail/" + id,
			"SubmitLabel": "Actualizar Personaje",
			"Form":        form,
			"Error":       "El nombre es obligatorio",
		})
		return
	}

	var existing models.Character
	if err := db.DB.Unscoped().Where("name = ? AND deleted_at IS NULL AND id != ?", form.Name, id).First(&existing).Error; err == nil {
		render(c, http.StatusConflict, "character-form.html", gin.H{
			"Title":       "Editar Personaje",
		"ActiveMenu":  "characters",
			"Action":      "/characters/detail/" + id,
			"SubmitLabel": "Actualizar Personaje",
			"Form":        form,
			"Error":       "Ya existe un personaje con ese nombre",
		})
		return
	}

	var character models.Character
	if err := db.DB.Unscoped().First(&character, id).Error; err != nil {
		c.Redirect(http.StatusFound, "/characters")
		return
	}

	character.Number = form.Number
	character.Player = form.Player
	character.Name = form.Name
	character.Status = form.Status
	character.Species = form.Species
	character.Class = form.Class
	character.GuildName = form.GuildName

	if err := db.DB.Save(&character).Error; err != nil {
		render(c, http.StatusInternalServerError, "character-form.html", gin.H{
			"Title":       "Editar Personaje",
		"ActiveMenu":  "characters",
			"Action":      "/characters/detail/" + id,
			"SubmitLabel": "Actualizar Personaje",
			"Form":        form,
			"Error":       "Error al actualizar el personaje",
		})
		return
	}

	c.Redirect(http.StatusFound, "/characters/detail/"+id)
}

func CharacterDeleteHandler(c *gin.Context) {
	id := c.Param("id")
	db.DB.Delete(&models.Character{}, id)
	c.Redirect(http.StatusFound, "/characters")
}

func SetupRoutes(r *gin.Engine) {
	r.Static("/static", "./static")
	r.LoadHTMLFiles("templates/login.html")

	r.GET("/login", LoginPageHandler)
	r.POST("/login", LoginPostHandler)

	auth := r.Group("/")
	auth.Use(AuthRequired(), CSRFRequired())
	{
		auth.POST("/logout", LogoutHandler)
		auth.GET("/", IndexHandler)
		auth.GET("/characters", CharactersHandler)
		auth.GET("/characters/create", CharacterNewHandler)
		auth.POST("/characters", CharacterCreateHandler)

		detail := auth.Group("/characters/detail")
		{
			detail.GET("/:id", CharacterDetailHandler)
			detail.GET("/:id/edit", CharacterEditHandler)
			detail.POST("/:id", CharacterUpdateHandler)
			detail.POST("/:id/delete", CharacterDeleteHandler)
		}
		auth.GET("/missions", MissionsHandler)
		auth.GET("/missions/create", MissionNewHandler)
		auth.POST("/missions", MissionCreateHandler)

		mDetail := auth.Group("/missions/detail")
		{
			mDetail.GET("/:id", MissionDetailHandler)
			mDetail.GET("/:id/edit", MissionEditHandler)
			mDetail.POST("/:id", MissionUpdateHandler)
			mDetail.POST("/:id/delete", MissionDeleteHandler)
			mDetail.POST("/:id/entries", MissionEntryCreateHandler)
			mDetail.GET("/:id/entries/:eid/edit", MissionEntryEditHandler)
			mDetail.POST("/:id/entries/:eid", MissionEntryUpdateHandler)
			mDetail.POST("/:id/entries/:eid/delete", MissionEntryDeleteHandler)
		}
		auth.GET("/dl", DLHandler)
		dlUsages := auth.Group("/dl/usages")
		{
			dlUsages.POST("", DLUsageCreateHandler)
			dlUsages.GET("/:id/edit", DLUsageEditHandler)
			dlUsages.POST("/:id", DLUsageUpdateHandler)
			dlUsages.POST("/:id/delete", DLUsageDeleteHandler)
		}
		auth.GET("/transactions", TransactionsHandler)
		auth.POST("/transactions", TransactionCreateHandler)

		txDetail := auth.Group("/transactions/detail")
		{
			txDetail.GET("/:id/edit", TransactionEditHandler)
			txDetail.POST("/:id", TransactionUpdateHandler)
			txDetail.POST("/:id/delete", TransactionDeleteHandler)
		}
		auth.GET("/guilds", GuildsHandler)
		auth.GET("/guilds/create", GuildNewHandler)
		auth.POST("/guilds", GuildCreateHandler)

		gDetail := auth.Group("/guilds/detail")
		{
			gDetail.GET("/:id", GuildDetailHandler)
			gDetail.GET("/:id/edit", GuildEditHandler)
			gDetail.POST("/:id", GuildUpdateHandler)
			gDetail.POST("/:id/delete", GuildDeleteHandler)
		}
	}
}

func parseUint(s string) uint {
	var v uint
	for _, c := range s {
		if c >= '0' && c <= '9' {
			v = v*10 + uint(c-'0')
		}
	}
	return v
}

func parseInt(s string) int {
	var v int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			v = v*10 + int(c-'0')
		}
	}
	return v
}

func parseFloat(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}
