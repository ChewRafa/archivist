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

	result := services.ImportExcel(f)

	errs := result.Errors
	success := true
	if len(errs) > 0 {
		success = false
	}

	summary := fmt.Sprintf(
		"Personajes: %d | Usos de DL: %d | Compras: %d | Costos de Vida: %d | Registros: %d | Misiones: %d | Entradas: %d | Gremios: %d",
		result.Characters, result.DLUsages, result.Transactions,
		result.CostOfLivings, result.Registries, result.Missions,
		result.MissionEntries, result.Guilds,
	)

	render(c, http.StatusOK, "import.html", gin.H{
		"Title":      "Importar Datos",
		"ActiveMenu": "import",
		"Result":     &result,
		"Summary":    summary,
		"Success":    success,
	})
}
