package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

func renderImportPage(c *gin.Context, status int, data gin.H) {
	if data == nil {
		data = gin.H{}
	}
	if _, ok := data["Title"]; !ok {
		data["Title"] = "Importar Datos"
	}
	if _, ok := data["ActiveMenu"]; !ok {
		data["ActiveMenu"] = "import"
	}
	if _, ok := data["Sheets"]; !ok {
		data["Sheets"] = AllSheetInfo()
	}
	if _, ok := data["Format"]; !ok {
		data["Format"] = ""
	}
	render(c, status, "import.html", data)
}

func ImportPageHandler(c *gin.Context) {
	renderImportPage(c, http.StatusOK, nil)
}

func ImportPostHandler(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		renderImportPage(c, http.StatusBadRequest, gin.H{
			"Error": "No se recibió ningún archivo",
		})
		return
	}

	if file.Size == 0 {
		renderImportPage(c, http.StatusBadRequest, gin.H{
			"Error": "El archivo está vacío",
		})
		return
	}

	src, err := file.Open()
	if err != nil {
		renderImportPage(c, http.StatusInternalServerError, gin.H{
			"Error": "Error al abrir el archivo",
		})
		return
	}
	defer src.Close()

	raw, err := io.ReadAll(src)
	if err != nil {
		renderImportPage(c, http.StatusInternalServerError, gin.H{
			"Error": "Error al leer el archivo",
		})
		return
	}

	switch detectImportFormat(file.Filename, raw) {
	case formatJSON:
		importJSONUpload(c, raw)
	case formatXLSX:
		importExcelUpload(c, raw)
	default:
		renderImportPage(c, http.StatusBadRequest, gin.H{
			"Error": "Formato no reconocido: se espera un archivo .xlsx o .json",
		})
	}
}

func importExcelUpload(c *gin.Context, raw []byte) {
	f, err := excelize.OpenReader(bytes.NewReader(raw))
	if err != nil {
		renderImportPage(c, http.StatusBadRequest, gin.H{
			"Error": "El archivo no es un Excel válido: " + err.Error(),
		})
		return
	}
	defer f.Close()

	selectedSheets := c.PostFormArray("sheets")
	var opts []ImportOptions
	if len(selectedSheets) > 0 {
		opts = append(opts, ImportOptions{Sheets: selectedSheets})
	}

	result := ImportExcel(f, opts...)

	summary := fmt.Sprintf(
		"Personajes: %d (%d omitidos) | Usos de DL: %d (%d omitidos) | Compras: %d (%d omitidos) | Costos de Vida: %d (%d omitidos) | Registros: %d (%d omitidos) | Misiones: %d (%d omitidos) | Entradas: %d (%d omitidos) | Gremios: %d (%d omitidos) | Economía de Gremios: %d (%d omitidas)",
		result.Characters, result.CharactersSkipped,
		result.DLUsages, result.DLUsagesSkipped,
		result.Transactions, result.TransactionsSkipped,
		result.CostOfLivings, result.CostOfLivingsSkipped,
		result.Registries, result.RegistriesSkipped,
		result.Missions, result.MissionsSkipped,
		result.MissionEntries, result.MissionEntriesSkipped,
		result.Guilds, result.GuildsSkipped,
		result.GuildTransactions, result.GuildTransactionsSkipped,
	)

	renderImportPage(c, http.StatusOK, gin.H{
		"Result":  &result,
		"Summary": summary,
		"Format":  formatXLSX,
	})
}

func importJSONUpload(c *gin.Context, raw []byte) {
	data, err := ValidateExportJSON(raw)
	if err != nil {
		renderImportPage(c, http.StatusBadRequest, gin.H{
			"Error": err.Error(),
		})
		return
	}

	if c.PostForm("restore_confirm") != "1" {
		renderImportPage(c, http.StatusBadRequest, gin.H{
			"Error": "Debes confirmar la restauración: este proceso borra todos los datos actuales",
		})
		return
	}

	result := ImportJSON(data)

	renderImportPage(c, http.StatusOK, gin.H{
		"Result":     &result,
		"Format":     formatJSON,
		"ExportMeta": BuildExportMeta(data),
	})
}
