package handlers

import (
	"fmt"
	"net/http"

	"codeberg.org/chewrafa/archivist/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

func ImportPageHandler(c *gin.Context) {
	render(c, http.StatusOK, "import.html", gin.H{
		"Title":      "Importar Datos",
		"ActiveMenu": "import",
		"Sheets":     services.AllSheetInfo(),
	})
}

func ImportPostHandler(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		render(c, http.StatusBadRequest, "import.html", gin.H{
			"Title":      "Importar Datos",
			"ActiveMenu": "import",
			"Error":      "No se recibió ningún archivo",
		})
		return
	}

	if file.Size == 0 {
		render(c, http.StatusBadRequest, "import.html", gin.H{
			"Title":      "Importar Datos",
			"ActiveMenu": "import",
			"Error":      "El archivo está vacío",
		})
		return
	}

	src, err := file.Open()
	if err != nil {
		render(c, http.StatusInternalServerError, "import.html", gin.H{
			"Title":      "Importar Datos",
			"ActiveMenu": "import",
			"Error":      "Error al abrir el archivo",
		})
		return
	}
	defer src.Close()

	f, err := excelize.OpenReader(src)
	if err != nil {
		render(c, http.StatusBadRequest, "import.html", gin.H{
			"Title":      "Importar Datos",
			"ActiveMenu": "import",
			"Error":      "El archivo no es un Excel válido: " + err.Error(),
		})
		return
	}
	defer f.Close()

	selectedSheets := c.PostFormArray("sheets")
	var opts []services.ImportOptions
	if len(selectedSheets) > 0 {
		opts = append(opts, services.ImportOptions{Sheets: selectedSheets})
	}

	result := services.ImportExcel(f, opts...)

	summary := fmt.Sprintf(
		"Personajes: %d (%d omitidos) | Usos de DL: %d (%d omitidos) | Compras: %d (%d omitidos) | Costos de Vida: %d (%d omitidos) | Registros: %d (%d omitidos) | Misiones: %d (%d omitidos) | Entradas: %d (%d omitidos) | Gremios: %d (%d omitidos)",
		result.Characters, result.CharactersSkipped,
		result.DLUsages, result.DLUsagesSkipped,
		result.Transactions, result.TransactionsSkipped,
		result.CostOfLivings, result.CostOfLivingsSkipped,
		result.Registries, result.RegistriesSkipped,
		result.Missions, result.MissionsSkipped,
		result.MissionEntries, result.MissionEntriesSkipped,
		result.Guilds, result.GuildsSkipped,
	)

	render(c, http.StatusOK, "import.html", gin.H{
		"Title":      "Importar Datos",
		"ActiveMenu": "import",
		"Result":     &result,
		"Summary":    summary,
	})
}
