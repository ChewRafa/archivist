package main

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type PageData struct {
	Title string
	Data  interface{}
}

var canonicalClasses = map[string]string{
	"artífice":  "Artífice",
	"bardo":     "Bardo",
	"brujo":     "Brujo",
	"bárbaro":   "Bárbaro",
	"clérigo":   "Clérigo",
	"druida":    "Druida",
	"explorador": "Explorador",
	"guerrero":  "Guerrero",
	"hechicero": "Hechicero",
	"mago":      "Mago",
	"monje":     "Monje",
	"paladín":   "Paladín",
	"pícaro":    "Pícaro",
	// Feminine variants
	"artífiza":   "Artífice",
	"barda":      "Bardo",
	"bruja":      "Brujo",
	"bárbara":    "Bárbaro",
	"clériga":    "Clérigo",
	"exploradora": "Explorador",
	"guerrera":   "Guerrero",
	"hechicera":  "Hechicero",
	"maga":       "Mago",
	"monja":      "Monje",
	"paladina":   "Paladín",
	"pícara":     "Pícaro",
}

func normalizeClassName(class string) string {
	if class == "" {
		return class
	}
	lower := strings.ToLower(class)
	if canonical, ok := canonicalClasses[lower]; ok {
		return canonical
	}
	return class
}

func IndexHandler(c *gin.Context) {
	stats, _ := GetAllCharactersWithStats()

	activeCount := 0
	// Mapas temporales para hacer el conteo bruto
	rawLevelDist := make(map[int]int)
	rawClassDist := make(map[string]int)
	rawSpeciesDist := make(map[string]int)

	for _, s := range stats {
		if s.Status == "Activo" {
			activeCount++
		}
		rawLevelDist[s.Level]++
		rawClassDist[normalizeClassName(s.Class)]++
		rawSpeciesDist[s.Species]++
	}

	// Estructura interna para empaquetar el conteo junto a su porcentaje lineal
	type ChartItem struct {
		Count      int
		Percentage float64
	}

	total := float64(len(stats))

	// Procesamos la distribuciÃ³n por Nivel
	levelDist := make(map[int]ChartItem)
	for lvl, count := range rawLevelDist {
		pct := 0.0
		if total > 0 {
			pct = (float64(count) / total) * 100
		}
		levelDist[lvl] = ChartItem{Count: count, Percentage: pct}
	}

	// Procesamos la distribuciÃ³n por Clase
	classDist := make(map[string]ChartItem)
	for class, count := range rawClassDist {
		pct := 0.0
		if total > 0 {
			pct = (float64(count) / total) * 100
		}
		classDist[class] = ChartItem{Count: count, Percentage: pct}
	}

	// Procesamos la distribuciÃ³n por Raza/Especie
	speciesDist := make(map[string]ChartItem)
	for species, count := range rawSpeciesDist {
		pct := 0.0
		if total > 0 {
			pct = (float64(count) / total) * 100
		}
		speciesDist[species] = ChartItem{Count: count, Percentage: pct}
	}

	// Tus consultas de GORM se quedan exactamente igual
	var recentMissions []Mission
	DB.Order("date DESC").Limit(5).Preload("Entries").Find(&recentMissions)

	var recentDLUsages []DLUsage
	DB.Order("date DESC").Limit(5).Preload("Character").Find(&recentDLUsages)

	var recentTransactions []Transaction
	DB.Order("date DESC").Limit(5).Preload("Character").Find(&recentTransactions)

	render(c, http.StatusOK, "index.html", gin.H{
		"Title":              "Dashboard",
		"ActiveMenu":         "dashboard",
		"Stats":              stats,
		"ActiveCount":        activeCount,
		"TotalCount":         len(stats),
		"LevelDist":          levelDist,   // Ahora envÃ­a el mapa con objetos ChartItem
		"ClassDist":          classDist,   // Ahora envÃ­a el mapa con objetos ChartItem
		"SpeciesDist":        speciesDist, // Ahora envÃ­a el mapa con objetos ChartItem
		"RecentMissions":     recentMissions,
		"RecentDLUsages":     recentDLUsages,
		"RecentTransactions": recentTransactions,
	})
}

func CharactersHandler(c *gin.Context) {
	stats, _ := GetAllCharactersWithStats()

	filter := c.Query("filter")
	if filter != "" {
		var filtered []CharacterStats
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
		"Title":      "Personajes",
		"ActiveMenu": "characters",
		"Stats":      stats,
		"Filter":     filter,
	})
}

func CharacterDetailHandler(c *gin.Context) {
	id := c.Param("id")
	stats, err := GetCharacterWithStats(parseUint(id))
	if err != nil {
		c.Redirect(http.StatusFound, "/characters")
		return
	}

	var registries []CharacterRegistry
	DB.Where("character_id = ?", id).Order("date DESC").Find(&registries)

	var missionEntries []MissionEntry
	DB.Where("character_id = ?", id).Order("created_at DESC").Preload("Mission").Find(&missionEntries)

	var dlUsages []DLUsage
	DB.Where("character_id = ?", id).Order("date DESC").Find(&dlUsages)

	var transactions []Transaction
	DB.Where("character_id = ?", id).Order("date DESC").Find(&transactions)

	var colEntries []CostOfLiving
	DB.Where("character_id = ?", id).Order("date DESC").Find(&colEntries)

	render(c, http.StatusOK, "character-detail.html", gin.H{
		"Title":          stats.Name,
		"ActiveMenu":     "characters",
		"Stats":          stats,
		"Registries":     registries,
		"MissionEntries": missionEntries,
		"DLUsages":       dlUsages,
		"Transactions":   transactions,
		"CostOfLivings":  colEntries,
	})
}

func MissionsHandler(c *gin.Context) {
	var missions []Mission
	DB.Order("date DESC").Preload("Entries").Preload("Entries.Character").Find(&missions)

	render(c, http.StatusOK, "missions.html", gin.H{
		"Title":      "Misiones",
		"ActiveMenu": "missions",
		"Missions":   missions,
	})
}

func DLHandler(c *gin.Context) {
	var usages []DLUsage
	DB.Order("date DESC").Preload("Character").Limit(50).Find(&usages)

	var characters []Character
	DB.Order("name ASC").Find(&characters)

	render(c, http.StatusOK, "dl.html", gin.H{
		"Title":      "Uso de DL",
		"ActiveMenu": "dl",
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
		DLUsed:      formInt(c.PostForm("dl_used")),
		GoldChange:  formFloat(c.PostForm("gold_change")),
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

	usage := DLUsage{
		Date:        date,
		CharacterID: form.CharacterID,
		DLUsed:      form.DLUsed,
		GoldChange:  form.GoldChange,
		Description: form.Description,
	}

	DB.Create(&usage)
	setFlash(c, "success", "Uso de DL registrado correctamente.")
	c.Redirect(http.StatusFound, "/dl")
}

func DLUsageEditHandler(c *gin.Context) {
	id := c.Param("id")
	var usage DLUsage
	if err := DB.First(&usage, id).Error; err != nil {
		c.Redirect(http.StatusFound, "/dl")
		return
	}

	var characters []Character
	DB.Order("name ASC").Find(&characters)

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
		DLUsed:      formInt(c.PostForm("dl_used")),
		GoldChange:  formFloat(c.PostForm("gold_change")),
		Description: c.PostForm("description"),
	}

	if form.Date == "" || form.CharacterID == 0 || form.DLUsed <= 0 {
		var characters []Character
		DB.Order("name ASC").Find(&characters)
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
		var characters []Character
		DB.Order("name ASC").Find(&characters)
		render(c, http.StatusBadRequest, "dl-usage-form.html", gin.H{
			"Title":       "Editar Uso de DL",
			"ActiveMenu":  "dl",
			"Action":      "/dl/usages/" + id,
			"SubmitLabel": "Actualizar",
			"Form":        form,
			"Characters":  characters,
			"Error":       "Fecha invÃ¡lida",
		})
		return
	}

	var usage DLUsage
	if err := DB.First(&usage, id).Error; err != nil {
		c.Redirect(http.StatusFound, "/dl")
		return
	}

	usage.Date = date
	usage.CharacterID = form.CharacterID
	usage.DLUsed = form.DLUsed
	usage.GoldChange = form.GoldChange
	usage.Description = form.Description

	if err := DB.Save(&usage).Error; err != nil {
		var characters []Character
		DB.Order("name ASC").Find(&characters)
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

	setFlash(c, "success", "Uso de DL actualizado correctamente.")
	c.Redirect(http.StatusFound, "/dl")
}

func DLUsageDeleteHandler(c *gin.Context) {
	id := c.Param("id")
	DB.Delete(&DLUsage{}, id)
	setFlash(c, "success", "Uso de DL eliminado.")
	c.Redirect(http.StatusFound, "/dl")
}

func TransactionsHandler(c *gin.Context) {
	var transactions []Transaction
	DB.Order("date DESC").Preload("Character").Limit(100).Find(&transactions)

	var characters []Character
	DB.Order("name ASC").Find(&characters)

	render(c, http.StatusOK, "transactions.html", gin.H{
		"Title":        "Transacciones",
		"ActiveMenu":   "transactions",
		"Transactions": transactions,
		"Characters":   characters,
	})
}

func TransactionCreateHandler(c *gin.Context) {
	date := c.PostForm("date")
	characterID := parseUint(c.PostForm("character_id"))
	amount := formFloat(c.PostForm("amount"))
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

	tx := Transaction{
		Date:        parsed,
		CharacterID: characterID,
		Amount:      amount,
		Notes:       notes,
	}

	DB.Create(&tx)
	setFlash(c, "success", "TransacciÃ³n aÃ±adida correctamente.")
	c.Redirect(http.StatusFound, "/transactions")
}

func TransactionEditHandler(c *gin.Context) {
	id := c.Param("id")
	var tx Transaction
	if err := DB.First(&tx, id).Error; err != nil {
		c.Redirect(http.StatusFound, "/transactions")
		return
	}

	var characters []Character
	DB.Order("name ASC").Find(&characters)

	render(c, http.StatusOK, "transaction-form.html", gin.H{
		"Title":       "Editar TransacciÃ³n",
		"ActiveMenu":  "transactions",
		"Action":      "/transactions/detail/" + id,
		"SubmitLabel": "Actualizar TransacciÃ³n",
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
		Amount:      formFloat(c.PostForm("amount")),
		Notes:       c.PostForm("notes"),
	}

	if form.Date == "" || form.CharacterID == 0 {
		var characters []Character
		DB.Order("name ASC").Find(&characters)
		render(c, http.StatusBadRequest, "transaction-form.html", gin.H{
			"Title":       "Editar TransacciÃ³n",
			"ActiveMenu":  "transactions",
			"Action":      "/transactions/detail/" + id,
			"SubmitLabel": "Actualizar TransacciÃ³n",
			"Form":        form,
			"Characters":  characters,
			"Error":       "Fecha y personaje son obligatorios",
		})
		return
	}

	date, err := time.Parse("2006-01-02", form.Date)
	if err != nil {
		var characters []Character
		DB.Order("name ASC").Find(&characters)
		render(c, http.StatusBadRequest, "transaction-form.html", gin.H{
			"Title":       "Editar TransacciÃ³n",
			"ActiveMenu":  "transactions",
			"Action":      "/transactions/detail/" + id,
			"SubmitLabel": "Actualizar TransacciÃ³n",
			"Form":        form,
			"Characters":  characters,
			"Error":       "Fecha invÃ¡lida",
		})
		return
	}

	var tx Transaction
	if err := DB.First(&tx, id).Error; err != nil {
		c.Redirect(http.StatusFound, "/transactions")
		return
	}

	tx.Date = date
	tx.CharacterID = form.CharacterID
	tx.Amount = form.Amount
	tx.Notes = form.Notes

	if err := DB.Save(&tx).Error; err != nil {
		var characters []Character
		DB.Order("name ASC").Find(&characters)
		render(c, http.StatusInternalServerError, "transaction-form.html", gin.H{
			"Title":       "Editar TransacciÃ³n",
			"ActiveMenu":  "transactions",
			"Action":      "/transactions/detail/" + id,
			"SubmitLabel": "Actualizar TransacciÃ³n",
			"Form":        form,
			"Characters":  characters,
			"Error":       "Error al actualizar la transacciÃ³n",
		})
		return
	}

	setFlash(c, "success", "TransacciÃ³n actualizada correctamente.")
	c.Redirect(http.StatusFound, "/transactions")
}

func TransactionDeleteHandler(c *gin.Context) {
	id := c.Param("id")
	DB.Delete(&Transaction{}, id)
	setFlash(c, "success", "TransacciÃ³n eliminada.")
	c.Redirect(http.StatusFound, "/transactions")
}

func GuildsHandler(c *gin.Context) {
	var guilds []Guild
	DB.Preload("Leader").Preload("Members").Find(&guilds)

	render(c, http.StatusOK, "guilds.html", gin.H{
		"Title":      "Gremios",
		"ActiveMenu": "guilds",
		"Guilds":     guilds,
	})
}

type GuildFormData struct {
	Name         string
	LeaderID     uint
	HallType     string
	Notes        string
	CostOfLiving float64
	RegisteredAt string
	ApprovedAt   string
}

type GuildTransactionFormData struct {
	Date   string
	Amount float64
	Notes  string
}

func GuildNewHandler(c *gin.Context) {
	var characters []Character
	DB.Order("name ASC").Find(&characters)

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
		CostOfLiving: formFloat(c.PostForm("cost_of_living")),
		RegisteredAt: c.PostForm("registered_at"),
		ApprovedAt:   c.PostForm("approved_at"),
	}

	if form.Name == "" {
		var characters []Character
		DB.Order("name ASC").Find(&characters)
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

	guild := Guild{
		Name:         form.Name,
		HallType:     form.HallType,
		Notes:        form.Notes,
		CostOfLiving: form.CostOfLiving,
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

	if err := DB.Create(&guild).Error; err != nil {
		var characters []Character
		DB.Order("name ASC").Find(&characters)
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

	setFlash(c, "success", "Gremio creado correctamente.")
	c.Redirect(http.StatusFound, "/guilds/detail/"+fmt.Sprint(guild.ID))
}

func GuildDetailHandler(c *gin.Context) {
	id := c.Param("id")
	var guild Guild
	if err := DB.Preload("Leader").Preload("Members").First(&guild, id).Error; err != nil {
		c.Redirect(http.StatusFound, "/guilds")
		return
	}

	var transactions []GuildTransaction
	DB.Where("guild_id = ?", guild.ID).Order("date DESC").Find(&transactions)

	render(c, http.StatusOK, "guild-detail.html", gin.H{
		"Title":        guild.Name,
		"ActiveMenu":   "guilds",
		"Guild":        guild,
		"Transactions": transactions,
	})
}

func GuildEditHandler(c *gin.Context) {
	id := c.Param("id")
	var guild Guild
	if err := DB.Preload("Leader").First(&guild, id).Error; err != nil {
		c.Redirect(http.StatusFound, "/guilds")
		return
	}

	var characters []Character
	DB.Order("name ASC").Find(&characters)

	form := GuildFormData{
		Name:         guild.Name,
		HallType:     guild.HallType,
		Notes:        guild.Notes,
		CostOfLiving: guild.CostOfLiving,
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
		"IsEdit":      true,
		"Treasury":    guild.Treasury,
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
		CostOfLiving: formFloat(c.PostForm("cost_of_living")),
		RegisteredAt: c.PostForm("registered_at"),
		ApprovedAt:   c.PostForm("approved_at"),
	}

	if form.Name == "" {
		var characters []Character
		DB.Order("name ASC").Find(&characters)
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

	var guild Guild
	if err := DB.First(&guild, id).Error; err != nil {
		c.Redirect(http.StatusFound, "/guilds")
		return
	}

	guild.Name = form.Name
	guild.HallType = form.HallType
	guild.Notes = form.Notes
	guild.CostOfLiving = form.CostOfLiving

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

	if err := DB.Save(&guild).Error; err != nil {
		var characters []Character
		DB.Order("name ASC").Find(&characters)
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

	setFlash(c, "success", "Gremio actualizado correctamente.")
	c.Redirect(http.StatusFound, "/guilds/detail/"+id)
}

func GuildDeleteHandler(c *gin.Context) {
	id := c.Param("id")
	DB.Where("guild_id = ?", id).Delete(&GuildTransaction{})
	DB.Delete(&Guild{}, id)
	setFlash(c, "success", "Gremio eliminado.")
	c.Redirect(http.StatusFound, "/guilds")
}

func GuildTransactionCreateHandler(c *gin.Context) {
	guildID := c.Param("id")
	date := c.PostForm("date")
	amount := formFloat(c.PostForm("amount"))
	notes := c.PostForm("notes")

	if date == "" {
		c.Redirect(http.StatusFound, "/guilds/detail/"+guildID)
		return
	}

	parsed, err := time.Parse("2006-01-02", date)
	if err != nil {
		c.Redirect(http.StatusFound, "/guilds/detail/"+guildID)
		return
	}

	gid := parseUint(guildID)
	entry := GuildTransaction{
		Date:    parsed,
		GuildID: gid,
		Amount:  amount,
		Notes:   notes,
	}

	DB.Transaction(func(tx *gorm.DB) error {
		_, err := CreateGuildTransaction(tx, entry)
		return err
	})

	setFlash(c, "success", "Movimiento de arcas aÃ±adido.")
	c.Redirect(http.StatusFound, "/guilds/detail/"+guildID)
}

func GuildTransactionEditHandler(c *gin.Context) {
	guildID := c.Param("id")
	txID := c.Param("txId")

	var entry GuildTransaction
	if err := DB.Where("id = ? AND guild_id = ?", txID, guildID).First(&entry).Error; err != nil {
		c.Redirect(http.StatusFound, "/guilds/detail/"+guildID)
		return
	}

	var guild Guild
	if err := DB.First(&guild, guildID).Error; err != nil {
		c.Redirect(http.StatusFound, "/guilds")
		return
	}

	render(c, http.StatusOK, "guild-transaction-form.html", gin.H{
		"Title":       "Editar Movimiento de Arcas",
		"ActiveMenu":  "guilds",
		"Guild":       guild,
		"Action":      "/guilds/detail/" + guildID + "/transactions/" + txID,
		"SubmitLabel": "Actualizar Movimiento",
		"Form": GuildTransactionFormData{
			Date:   entry.Date.Format("2006-01-02"),
			Amount: entry.Amount,
			Notes:  entry.Notes,
		},
		"Error": "",
	})
}

func GuildTransactionUpdateHandler(c *gin.Context) {
	guildID := c.Param("id")
	txID := c.Param("txId")
	form := GuildTransactionFormData{
		Date:   c.PostForm("date"),
		Amount: formFloat(c.PostForm("amount")),
		Notes:  c.PostForm("notes"),
	}

	var guild Guild
	if err := DB.First(&guild, guildID).Error; err != nil {
		c.Redirect(http.StatusFound, "/guilds")
		return
	}

	if form.Date == "" {
		render(c, http.StatusBadRequest, "guild-transaction-form.html", gin.H{
			"Title":       "Editar Movimiento de Arcas",
			"ActiveMenu":  "guilds",
			"Guild":       guild,
			"Action":      "/guilds/detail/" + guildID + "/transactions/" + txID,
			"SubmitLabel": "Actualizar Movimiento",
			"Form":        form,
			"Error":       "La fecha es obligatoria",
		})
		return
	}

	date, err := time.Parse("2006-01-02", form.Date)
	if err != nil {
		render(c, http.StatusBadRequest, "guild-transaction-form.html", gin.H{
			"Title":       "Editar Movimiento de Arcas",
			"ActiveMenu":  "guilds",
			"Guild":       guild,
			"Action":      "/guilds/detail/" + guildID + "/transactions/" + txID,
			"SubmitLabel": "Actualizar Movimiento",
			"Form":        form,
			"Error":       "Fecha invÃ¡lida",
		})
		return
	}

	var entry GuildTransaction
	if err := DB.Where("id = ? AND guild_id = ?", txID, guildID).First(&entry).Error; err != nil {
		c.Redirect(http.StatusFound, "/guilds/detail/"+guildID)
		return
	}

	entry.Date = date
	entry.Amount = form.Amount
	entry.Notes = form.Notes

	DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&entry).Error; err != nil {
			return err
		}
		return SyncGuildTreasury(tx, entry.GuildID)
	})

	setFlash(c, "success", "Movimiento de arcas actualizado.")
	c.Redirect(http.StatusFound, "/guilds/detail/"+guildID)
}

func GuildTransactionDeleteHandler(c *gin.Context) {
	guildID := c.Param("id")
	txID := c.Param("txId")

	var entry GuildTransaction
	if err := DB.Where("id = ? AND guild_id = ?", txID, guildID).First(&entry).Error; err != nil {
		c.Redirect(http.StatusFound, "/guilds/detail/"+guildID)
		return
	}

	DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&entry).Error; err != nil {
			return err
		}
		return SyncGuildTreasury(tx, entry.GuildID)
	})

	setFlash(c, "success", "Movimiento de arcas eliminado.")
	c.Redirect(http.StatusFound, "/guilds/detail/"+guildID)
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
		"Title":       "Nueva MisiÃ³n",
		"ActiveMenu":  "missions",
		"Action":      "/missions",
		"SubmitLabel": "Crear MisiÃ³n",
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
			"Title":       "Nueva MisiÃ³n",
			"ActiveMenu":  "missions",
			"Action":      "/missions",
			"SubmitLabel": "Crear MisiÃ³n",
			"Form":        form,
			"Error":       "El nombre y el DM son obligatorios",
		})
		return
	}

	date, err := time.Parse("2006-01-02", form.Date)
	if err != nil {
		render(c, http.StatusBadRequest, "mission-form.html", gin.H{
			"Title":       "Nueva MisiÃ³n",
			"ActiveMenu":  "missions",
			"Action":      "/missions",
			"SubmitLabel": "Crear MisiÃ³n",
			"Form":        form,
			"Error":       "Fecha invÃ¡lida",
		})
		return
	}

	mission := Mission{
		Date: date,
		DM:   form.DM,
		Name: form.Name,
	}

	if err := DB.Create(&mission).Error; err != nil {
		render(c, http.StatusInternalServerError, "mission-form.html", gin.H{
			"Title":       "Nueva MisiÃ³n",
			"ActiveMenu":  "missions",
			"Action":      "/missions",
			"SubmitLabel": "Crear MisiÃ³n",
			"Form":        form,
			"Error":       "Error al crear la misiÃ³n",
		})
		return
	}

	setFlash(c, "success", "MisiÃ³n creada correctamente.")
	c.Redirect(http.StatusFound, "/missions/detail/"+fmt.Sprint(mission.ID))
}

func MissionDetailHandler(c *gin.Context) {
	id := c.Param("id")
	var mission Mission
	if err := DB.Preload("Entries", func(db *gorm.DB) *gorm.DB {
		return db.Where("deleted_at IS NULL").Order("created_at ASC")
	}).Preload("Entries.Character").First(&mission, id).Error; err != nil {
		c.Redirect(http.StatusFound, "/missions")
		return
	}

	var characters []Character
	DB.Order("name ASC").Find(&characters)

	render(c, http.StatusOK, "mission-detail.html", gin.H{
		"Title":      mission.Name,
		"ActiveMenu": "missions",
		"Mission":    mission,
		"Characters": characters,
		"EntryForm":  MissionEntryFormData{},
		"Error":      "",
	})
}

func MissionEditHandler(c *gin.Context) {
	id := c.Param("id")
	var mission Mission
	if err := DB.First(&mission, id).Error; err != nil {
		c.Redirect(http.StatusFound, "/missions")
		return
	}

	render(c, http.StatusOK, "mission-form.html", gin.H{
		"Title":       "Editar MisiÃ³n",
		"ActiveMenu":  "missions",
		"Action":      "/missions/detail/" + id,
		"SubmitLabel": "Actualizar MisiÃ³n",
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
			"Title":       "Editar MisiÃ³n",
			"ActiveMenu":  "missions",
			"Action":      "/missions/detail/" + id,
			"SubmitLabel": "Actualizar MisiÃ³n",
			"Form":        form,
			"Error":       "El nombre y el DM son obligatorios",
		})
		return
	}

	date, err := time.Parse("2006-01-02", form.Date)
	if err != nil {
		render(c, http.StatusBadRequest, "mission-form.html", gin.H{
			"Title":       "Editar MisiÃ³n",
			"ActiveMenu":  "missions",
			"Action":      "/missions/detail/" + id,
			"SubmitLabel": "Actualizar MisiÃ³n",
			"Form":        form,
			"Error":       "Fecha invÃ¡lida",
		})
		return
	}

	var mission Mission
	if err := DB.First(&mission, id).Error; err != nil {
		c.Redirect(http.StatusFound, "/missions")
		return
	}

	mission.Date = date
	mission.DM = form.DM
	mission.Name = form.Name

	if err := DB.Save(&mission).Error; err != nil {
		render(c, http.StatusInternalServerError, "mission-form.html", gin.H{
			"Title":       "Editar MisiÃ³n",
			"ActiveMenu":  "missions",
			"Action":      "/missions/detail/" + id,
			"SubmitLabel": "Actualizar MisiÃ³n",
			"Form":        form,
			"Error":       "Error al actualizar la misiÃ³n",
		})
		return
	}

	setFlash(c, "success", "MisiÃ³n actualizada correctamente.")
	c.Redirect(http.StatusFound, "/missions/detail/"+id)
}

func MissionDeleteHandler(c *gin.Context) {
	id := c.Param("id")

	var entries []MissionEntry
	DB.Where("mission_id = ?", id).Find(&entries)
	for _, entry := range entries {
		DB.Delete(&entry)
	}

	DB.Delete(&Mission{}, id)
	setFlash(c, "success", "MisiÃ³n eliminada.")
	c.Redirect(http.StatusFound, "/missions")
}

func MissionEntryCreateHandler(c *gin.Context) {
	id := c.Param("id")

	var mission Mission
	if err := DB.First(&mission, id).Error; err != nil {
		c.Redirect(http.StatusFound, "/missions")
		return
	}

	entry := MissionEntry{
		MissionID:   mission.ID,
		CharacterID: parseUint(c.PostForm("character_id")),
		XPMission:   formFloat(c.PostForm("xp_mission")),
		XPReport:    formFloat(c.PostForm("xp_report")),
		XPGuild:     formFloat(c.PostForm("xp_guild")),
		Gold:        formFloat(c.PostForm("gold")),
		Renown:      formFloat(c.PostForm("renown")),
		Notes:       c.PostForm("notes"),
	}

	DB.Create(&entry)
	setFlash(c, "success", "Personaje aÃ±adido a la misiÃ³n.")
	c.Redirect(http.StatusFound, "/missions/detail/"+id)
}

func MissionEntryEditHandler(c *gin.Context) {
	id := c.Param("id")
	eid := c.Param("eid")

	var mission Mission
	if err := DB.First(&mission, id).Error; err != nil {
		c.Redirect(http.StatusFound, "/missions")
		return
	}

	var entry MissionEntry
	if err := DB.Preload("Character").First(&entry, eid).Error; err != nil {
		c.Redirect(http.StatusFound, "/missions/detail/"+id)
		return
	}

	var characters []Character
	DB.Order("name ASC").Find(&characters)

	render(c, http.StatusOK, "mission-entry-form.html", gin.H{
		"Title":      "Editar Entrada de MisiÃ³n",
		"ActiveMenu": "missions",
		"Mission":    mission,
		"Entry":      entry,
		"Characters": characters,
		"Error":      "",
	})
}

func MissionEntryUpdateHandler(c *gin.Context) {
	id := c.Param("id")
	eid := c.Param("eid")

	var entry MissionEntry
	if err := DB.First(&entry, eid).Error; err != nil {
		c.Redirect(http.StatusFound, "/missions/detail/"+id)
		return
	}

	entry.CharacterID = parseUint(c.PostForm("character_id"))
	entry.XPMission = formFloat(c.PostForm("xp_mission"))
	entry.XPReport = formFloat(c.PostForm("xp_report"))
	entry.XPGuild = formFloat(c.PostForm("xp_guild"))
	entry.Gold = formFloat(c.PostForm("gold"))
	entry.Renown = formFloat(c.PostForm("renown"))
	entry.Notes = c.PostForm("notes")

	DB.Save(&entry)
	setFlash(c, "success", "Entrada actualizada correctamente.")
	c.Redirect(http.StatusFound, "/missions/detail/"+id)
}

func MissionEntryDeleteHandler(c *gin.Context) {
	id := c.Param("id")
	eid := c.Param("eid")
	DB.Delete(&MissionEntry{}, eid)
	setFlash(c, "success", "Entrada eliminada.")
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
	GuildRole string
	Mount     string
}

type CostOfLivingFormData struct {
	Date        string
	CharacterID uint
	Amount      float64
	Notes       string
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
		Number:    formInt(c.PostForm("number")),
		Player:    c.PostForm("player"),
		Name:      c.PostForm("name"),
		Status:    c.PostForm("status"),
		Species:   c.PostForm("species"),
		Class:     c.PostForm("class"),
		GuildName: c.PostForm("guild_name"),
		GuildRole: c.PostForm("guild_role"),
		Mount:     c.PostForm("mount"),
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

	var existing Character
	if err := DB.Unscoped().Where("name = ? AND deleted_at IS NULL", form.Name).First(&existing).Error; err == nil {
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

	character := Character{
		Number:    form.Number,
		Player:    form.Player,
		Name:      form.Name,
		Status:    form.Status,
		Species:   form.Species,
		Class:     form.Class,
		GuildName: form.GuildName,
		GuildRole: form.GuildRole,
		Mount:     form.Mount,
	}

	if err := DB.Create(&character).Error; err != nil {
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

	setFlash(c, "success", "Personaje creado correctamente.")
	c.Redirect(http.StatusFound, "/characters/detail/"+fmt.Sprint(character.ID))
}

func CharacterEditHandler(c *gin.Context) {
	id := c.Param("id")
	var character Character
	if err := DB.Unscoped().First(&character, id).Error; err != nil {
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
			GuildRole: character.GuildRole,
			Mount:     character.Mount,
		},
		"Error": "",
	})
}

func CharacterUpdateHandler(c *gin.Context) {
	id := c.Param("id")

	form := FormData{
		Number:    formInt(c.PostForm("number")),
		Player:    c.PostForm("player"),
		Name:      c.PostForm("name"),
		Status:    c.PostForm("status"),
		Species:   c.PostForm("species"),
		Class:     c.PostForm("class"),
		GuildName: c.PostForm("guild_name"),
		GuildRole: c.PostForm("guild_role"),
		Mount:     c.PostForm("mount"),
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

	var existing Character
	if err := DB.Unscoped().Where("name = ? AND deleted_at IS NULL AND id != ?", form.Name, id).First(&existing).Error; err == nil {
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

	var character Character
	if err := DB.Unscoped().First(&character, id).Error; err != nil {
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
	character.GuildRole = form.GuildRole
	character.Mount = form.Mount

	if err := DB.Save(&character).Error; err != nil {
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

	setFlash(c, "success", "Personaje actualizado correctamente.")
	c.Redirect(http.StatusFound, "/characters/detail/"+id)
}

func CharacterDeleteHandler(c *gin.Context) {
	id := c.Param("id")
	DB.Delete(&Character{}, id)
	setFlash(c, "success", "Personaje eliminado.")
	c.Redirect(http.StatusFound, "/characters")
}

func CostOfLivingHandler(c *gin.Context) {
	var entries []CostOfLiving
	DB.Order("date DESC").Preload("Character").Limit(100).Find(&entries)

	var characters []Character
	DB.Order("name ASC").Find(&characters)

	render(c, http.StatusOK, "cost-of-living.html", gin.H{
		"Title":      "Costo de Vida",
		"ActiveMenu": "cost-of-living",
		"Entries":    entries,
		"Characters": characters,
	})
}

func CostOfLivingCreateHandler(c *gin.Context) {
	form := CostOfLivingFormData{
		Date:        c.PostForm("date"),
		CharacterID: parseUint(c.PostForm("character_id")),
		Amount:      formFloat(c.PostForm("amount")),
		Notes:       c.PostForm("notes"),
	}

	if form.Date == "" || form.CharacterID == 0 {
		c.Redirect(http.StatusFound, "/cost-of-living")
		return
	}

	date, err := time.Parse("2006-01-02", form.Date)
	if err != nil {
		c.Redirect(http.StatusFound, "/cost-of-living")
		return
	}

	entry := CostOfLiving{
		Date:        date,
		CharacterID: form.CharacterID,
		Amount:      form.Amount,
		Notes:       form.Notes,
	}

	DB.Create(&entry)
	setFlash(c, "success", "Costo de vida registrado.")
	c.Redirect(http.StatusFound, "/cost-of-living")
}

func CostOfLivingEditHandler(c *gin.Context) {
	id := c.Param("id")
	var entry CostOfLiving
	if err := DB.First(&entry, id).Error; err != nil {
		c.Redirect(http.StatusFound, "/cost-of-living")
		return
	}

	var characters []Character
	DB.Order("name ASC").Find(&characters)

	render(c, http.StatusOK, "cost-of-living-form.html", gin.H{
		"Title":       "Editar Costo de Vida",
		"ActiveMenu":  "cost-of-living",
		"Action":      "/cost-of-living/" + id,
		"SubmitLabel": "Actualizar",
		"Form": CostOfLivingFormData{
			Date:        entry.Date.Format("2006-01-02"),
			CharacterID: entry.CharacterID,
			Amount:      entry.Amount,
			Notes:       entry.Notes,
		},
		"Characters": characters,
		"Error":      "",
	})
}

func CostOfLivingUpdateHandler(c *gin.Context) {
	id := c.Param("id")
	form := CostOfLivingFormData{
		Date:        c.PostForm("date"),
		CharacterID: parseUint(c.PostForm("character_id")),
		Amount:      formFloat(c.PostForm("amount")),
		Notes:       c.PostForm("notes"),
	}

	if form.Date == "" || form.CharacterID == 0 {
		var characters []Character
		DB.Order("name ASC").Find(&characters)
		render(c, http.StatusBadRequest, "cost-of-living-form.html", gin.H{
			"Title":       "Editar Costo de Vida",
			"ActiveMenu":  "cost-of-living",
			"Action":      "/cost-of-living/" + id,
			"SubmitLabel": "Actualizar",
			"Form":        form,
			"Characters":  characters,
			"Error":       "Fecha y personaje son obligatorios",
		})
		return
	}

	date, err := time.Parse("2006-01-02", form.Date)
	if err != nil {
		var characters []Character
		DB.Order("name ASC").Find(&characters)
		render(c, http.StatusBadRequest, "cost-of-living-form.html", gin.H{
			"Title":       "Editar Costo de Vida",
			"ActiveMenu":  "cost-of-living",
			"Action":      "/cost-of-living/" + id,
			"SubmitLabel": "Actualizar",
			"Form":        form,
			"Characters":  characters,
			"Error":       "Fecha invÃ¡lida",
		})
		return
	}

	var entry CostOfLiving
	if err := DB.First(&entry, id).Error; err != nil {
		c.Redirect(http.StatusFound, "/cost-of-living")
		return
	}

	entry.Date = date
	entry.CharacterID = form.CharacterID
	entry.Amount = form.Amount
	entry.Notes = form.Notes

	if err := DB.Save(&entry).Error; err != nil {
		var characters []Character
		DB.Order("name ASC").Find(&characters)
		render(c, http.StatusInternalServerError, "cost-of-living-form.html", gin.H{
			"Title":       "Editar Costo de Vida",
			"ActiveMenu":  "cost-of-living",
			"Action":      "/cost-of-living/" + id,
			"SubmitLabel": "Actualizar",
			"Form":        form,
			"Characters":  characters,
			"Error":       "Error al actualizar",
		})
		return
	}

	setFlash(c, "success", "Costo de vida actualizado.")
	c.Redirect(http.StatusFound, "/cost-of-living")
}

func CostOfLivingDeleteHandler(c *gin.Context) {
	id := c.Param("id")
	DB.Delete(&CostOfLiving{}, id)
	setFlash(c, "success", "Registro de costo de vida eliminado.")
	c.Redirect(http.StatusFound, "/cost-of-living")
}

func SetupRoutes(r *gin.Engine) {
	r.Static("/static", "./resources/static")
	r.LoadHTMLFiles("resources/login.html")

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
		auth.GET("/cost-of-living", CostOfLivingHandler)
		auth.POST("/cost-of-living", CostOfLivingCreateHandler)
		colDetail := auth.Group("/cost-of-living")
		{
			colDetail.GET("/:id/edit", CostOfLivingEditHandler)
			colDetail.POST("/:id", CostOfLivingUpdateHandler)
			colDetail.POST("/:id/delete", CostOfLivingDeleteHandler)
		}
		auth.GET("/import", ImportPageHandler)
		auth.POST("/import", ImportPostHandler)
		auth.GET("/guilds", GuildsHandler)
		auth.GET("/guilds/create", GuildNewHandler)
		auth.POST("/guilds", GuildCreateHandler)

		gDetail := auth.Group("/guilds/detail")
		{
			gDetail.GET("/:id", GuildDetailHandler)
			gDetail.GET("/:id/edit", GuildEditHandler)
			gDetail.POST("/:id", GuildUpdateHandler)
			gDetail.POST("/:id/delete", GuildDeleteHandler)
			gDetail.POST("/:id/transactions", GuildTransactionCreateHandler)
			gDetail.GET("/:id/transactions/:txId/edit", GuildTransactionEditHandler)
			gDetail.POST("/:id/transactions/:txId", GuildTransactionUpdateHandler)
			gDetail.POST("/:id/transactions/:txId/delete", GuildTransactionDeleteHandler)
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

func formInt(s string) int {
	var v int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			v = v*10 + int(c-'0')
		}
	}
	return v
}

func formFloat(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}
