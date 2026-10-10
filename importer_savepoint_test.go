//go:build !importer

package main

import (
	"testing"

	"gorm.io/gorm"
)

func TestImportSavepointKeepsTransactionUsableAfterInsertError(t *testing.T) {
	initTestDB(t)

	var dupErr error
	err := DB.Transaction(func(tx *gorm.DB) error {
		first := Character{Number: 1, Player: "Ada", Name: "First", Status: "Activo"}
		if err := withImportSavepoint(tx, func(tx *gorm.DB) error {
			return tx.Create(&first).Error
		}); err != nil {
			return err
		}

		dup := Character{Number: 1, Player: "Bea", Name: "Duplicate", Status: "Activo"}
		dupErr = withImportSavepoint(tx, func(tx *gorm.DB) error {
			return tx.Create(&dup).Error
		})

		third := Character{Number: 2, Player: "Cy", Name: "Third", Status: "Activo"}
		return withImportSavepoint(tx, func(tx *gorm.DB) error {
			return tx.Create(&third).Error
		})
	})
	if err != nil {
		t.Fatalf("transaction: %s", err)
	}
	if dupErr == nil {
		t.Fatal("expected unique constraint error")
	}

	var names []string
	if err := DB.Model(&Character{}).Order("number").Pluck("name", &names).Error; err != nil {
		t.Fatalf("list characters: %s", err)
	}
	if len(names) != 2 || names[0] != "First" || names[1] != "Third" {
		t.Fatalf("characters = %v, duplicate error = %v", names, dupErr)
	}
}
