package database

import (
	"testing"
)

func TestInitDB_Memory(t *testing.T) {
	db, err := InitDB(":memory:")
	if err != nil {
		t.Fatalf("failed to init in-memory db: %v", err)
	}
	defer db.Close()

	// Verify languages table populated
	var langCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM languages`).Scan(&langCount)
	if err != nil {
		t.Fatalf("failed to count languages: %v", err)
	}
	if langCount < 5 {
		t.Errorf("expected at least 5 seeded languages, got %d", langCount)
	}

	// Verify presets table populated
	var presetCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM presets`).Scan(&presetCount)
	if err != nil {
		t.Fatalf("failed to count presets: %v", err)
	}
	if presetCount < 3 {
		t.Errorf("expected at least 3 seeded presets, got %d", presetCount)
	}

	// Verify vocabularies table populated
	var vocabCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM vocabularies`).Scan(&vocabCount)
	if err != nil {
		t.Fatalf("failed to count vocabularies: %v", err)
	}
	if vocabCount < 10 {
		t.Errorf("expected at least 10 seeded vocabularies, got %d", vocabCount)
	}
}

