package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

// InitDB initializes SQLite connection with WAL mode and runs migrations
func InitDB(dbPath string) (*sql.DB, error) {
	if dbPath == "" {
		dbPath = os.Getenv("DATABASE_PATH")
		if dbPath == "" {
			dbPath = "lingua.db"
		}
	}

	// Ensure directory exists if path has subdirs
	dir := filepath.Dir(dbPath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create db directory: %w", err)
		}
	}

	// DSN with WAL mode and busy_timeout for concurrency
	dsn := fmt.Sprintf("%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=synchronous(NORMAL)", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping sqlite database: %w", err)
	}

	DB = db

	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	if err := SeedData(db); err != nil {
		log.Printf("[DB] Warning: seeding failed or already populated: %v", err)
	}

	log.Printf("[DB] SQLite initialized successfully (WAL mode enabled): %s", dbPath)
	return db, nil
}

func runMigrations(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS languages (
		code TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		native_name TEXT NOT NULL,
		flag TEXT NOT NULL,
		voice_name TEXT NOT NULL,
		pitch REAL DEFAULT 1.0,
		rate REAL DEFAULT 1.0
	);

	CREATE TABLE IF NOT EXISTS vocabularies (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		lang_code TEXT NOT NULL,
		word TEXT NOT NULL,
		clean_word TEXT NOT NULL,
		ipa TEXT NOT NULL,
		pos TEXT NOT NULL,
		meaning TEXT NOT NULL,
		example TEXT DEFAULT '',
		example_translation TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(lang_code, clean_word)
	);
	CREATE INDEX IF NOT EXISTS idx_vocab_lookup ON vocabularies(lang_code, clean_word);

	CREATE TABLE IF NOT EXISTS presets (
		id TEXT PRIMARY KEY,
		lang_code TEXT NOT NULL,
		category TEXT NOT NULL,
		title TEXT NOT NULL,
		description TEXT NOT NULL,
		content TEXT NOT NULL,
		difficulty TEXT NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_presets_lang ON presets(lang_code);

	CREATE TABLE IF NOT EXISTS bookmarks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		lang_code TEXT NOT NULL,
		text TEXT NOT NULL,
		ipa TEXT DEFAULT '',
		meaning TEXT DEFAULT '',
		context TEXT DEFAULT '',
		notes TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_bookmarks_lang ON bookmarks(lang_code);
	`

	_, err := db.Exec(schema)
	return err
}

