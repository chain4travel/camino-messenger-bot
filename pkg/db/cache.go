package db

import (
	"database/sql"
	"fmt"
	"log"
	"math/big"

	_ "github.com/mattn/go-sqlite3"
)

func InitDB(filepath string) *sql.DB {
	db, err := sql.Open("sqlite3", filepath)
	if err != nil {
		log.Fatal(err)
	}
	createTableSQL := `CREATE TABLE IF NOT EXISTS cheques (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    from_bot TEXT NOT NULL,
    to_bot TEXT NOT NULL,
	amount INTEGER NOT NULL,
    counter INTEGER NOT NULL,
    created_at INTEGER NOT NULL
	);`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatalf("%q: %s\n", err, createTableSQL)
	}
	return db
}

func InsertCheque(db *sql.DB, fromBot string, toBot string, amount big.Int, counter int64, createdAt int64) {
	query := `
	INSERT INTO cheques (from_bot, to_bot, amount, counter, created_at)
	VALUES (?, ?, ?, ?);`
	_, err := db.Exec(query, fromBot, toBot, counter, createdAt)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("New cheque from %s to %s inserted with amount: %d and counter: %d\n", fromBot, toBot, amount, counter)
}

func GetLatestCounter(db *sql.DB, fromBot string, toBot string) (int64, error) {
	query := `SELECT counter FROM cheques WHERE from_bot = ? AND to_bot = ? ORDER BY created_at DESC LIMIT 1;`
	var counter int64
	err := db.QueryRow(query, fromBot, toBot).Scan(&counter)
	if err != nil {
		if err == sql.ErrNoRows {
			// If no row is found, return 0 as the default counter
			return 0, nil
		}
		return 0, err
	}
	return counter, nil
}
