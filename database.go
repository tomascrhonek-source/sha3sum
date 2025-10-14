package main

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

func dbConnect(cfg config) *sql.DB {
	connStr := fmt.Sprintf("user=%s dbname=%s host=%s password='%s' port=%d sslmode=disable",
		cfg.dbUser, cfg.dbName, cfg.dbHost, cfg.dbPassword, cfg.dbPort)
	if *cfg.logging {
		log.Println("Connecting to database:", connStr)
	}
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	// Test connection
	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	return db
}

func saveToDB(db *sql.DB, logging *bool) {
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	defer tx.Commit()

	pool.Range(func(key any, value any) bool {
		_, err := tx.Exec("INSERT INTO sha3sum (path, sum, size) VALUES ($1, $2, $3)", value.(entry).path, hex.EncodeToString(value.(entry).hash), value.(entry).size)
		if err != nil {
			log.Println("Error:", err)
			tx.Rollback()
		}
		if *logging {
			log.Println("Inserted:", value.(entry).path)
		}

		if *logging {
			log.Println("All entries inserted")
		}
		return true
	})
}
