//go:build !importer

package main

import (
	"path/filepath"
	"testing"
)

func initTestDB(t *testing.T) {
	t.Helper()
	Init(filepath.Join(t.TempDir(), "test.db"))
	t.Cleanup(func() {
		sqlDB, err := DB.DB()
		if err != nil {
			return
		}
		sqlDB.Close()
	})
}

func TestUpsertAdminUserCreates(t *testing.T) {
	initTestDB(t)

	created, err := UpsertAdminUser("admin", "secret123")
	if err != nil {
		t.Fatalf("UpsertAdminUser: %s", err)
	}
	if !created {
		t.Fatal("expected created=true on empty DB")
	}
	if _, err := Authenticate("admin", "secret123"); err != nil {
		t.Fatalf("Authenticate after create: %s", err)
	}
}

func TestUpsertAdminUserResetsPassword(t *testing.T) {
	initTestDB(t)

	if _, err := UpsertAdminUser("admin", "oldpass"); err != nil {
		t.Fatalf("initial UpsertAdminUser: %s", err)
	}
	created, err := UpsertAdminUser("admin", "newpass")
	if err != nil {
		t.Fatalf("second UpsertAdminUser: %s", err)
	}
	if created {
		t.Fatal("expected created=false when user already exists")
	}
	if _, err := Authenticate("admin", "newpass"); err != nil {
		t.Fatalf("Authenticate with new password: %s", err)
	}
	if _, err := Authenticate("admin", "oldpass"); err == nil {
		t.Fatal("old password should no longer work")
	}
}

func TestUpsertAdminUserRejectsEmpty(t *testing.T) {
	initTestDB(t)

	if _, err := UpsertAdminUser("", "secret123"); err == nil {
		t.Fatal("expected error for empty username")
	}
	if _, err := UpsertAdminUser("admin", ""); err == nil {
		t.Fatal("expected error for empty password")
	}
}

func TestEnsureAdminFromEnvCreatesAndSyncs(t *testing.T) {
	initTestDB(t)

	t.Setenv("ADMIN_USERNAME", "envadmin")
	t.Setenv("ADMIN_PASSWORD", "firstpass")
	t.Setenv("DATABASE_URL", "")

	EnsureAdminFromEnv()
	if _, err := Authenticate("envadmin", "firstpass"); err != nil {
		t.Fatalf("Authenticate after seed: %s", err)
	}

	t.Setenv("ADMIN_PASSWORD", "rotatedpass")
	EnsureAdminFromEnv()
	if _, err := Authenticate("envadmin", "rotatedpass"); err != nil {
		t.Fatalf("Authenticate after password sync: %s", err)
	}
}

func TestEnsureAdminFromEnvCreatesAlongsideExistingUsers(t *testing.T) {
	initTestDB(t)

	if err := CreateUser("other", "otherpass"); err != nil {
		t.Fatalf("CreateUser: %s", err)
	}
	t.Setenv("ADMIN_USERNAME", "envadmin")
	t.Setenv("ADMIN_PASSWORD", "envpass")
	t.Setenv("DATABASE_URL", "")

	EnsureAdminFromEnv()
	if _, err := Authenticate("envadmin", "envpass"); err != nil {
		t.Fatalf("Authenticate seeded admin: %s", err)
	}
	if _, err := Authenticate("other", "otherpass"); err != nil {
		t.Fatalf("pre-existing user broken by seed: %s", err)
	}
}

func TestEnsureAdminFromEnvMissingNoop(t *testing.T) {
	initTestDB(t)

	t.Setenv("ADMIN_USERNAME", "")
	t.Setenv("ADMIN_PASSWORD", "")

	EnsureAdminFromEnv()

	var count int64
	if err := DB.Model(&User{}).Count(&count).Error; err != nil {
		t.Fatalf("Count: %s", err)
	}
	if count != 0 {
		t.Fatalf("expected no users created, got %d", count)
	}
}
