//go:build testprograms

package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	// Test database connection with same parameters as HelixCode
	dsn := "postgres://helixcode:helixcode123@localhost:5432/helixcode?sslmode=disable"
	
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Test connection
	err = db.Ping()
	if err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	fmt.Println("✅ Database connection successful!")

	// Test query
	var result string
	err = db.QueryRow("SELECT current_user").Scan(&result)
	if err != nil {
		log.Fatalf("Failed to query: %v", err)
	}

	fmt.Printf("✅ Current user: %s\n", result)
}