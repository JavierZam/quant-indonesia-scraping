package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"

	"github.com/JavierZam/quant-indonesia-scraping/pkg/sector"
)

func main() {
	_ = godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://quantuser:quantpass@localhost:5432/quantintel?sslmode=disable"
	}

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		fmt.Printf("❌ Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(ctx)

	fmt.Println("⚡ Connected to PostgreSQL!")
	fmt.Println("🔧 Fixing misclassified stock sectors using official IDX sector map...")

	var updatedCount int
	for symbol, officialSector := range sector.OfficialIDXMap {
		tag, err := conn.Exec(ctx, `UPDATE stocks SET sector = $1 WHERE symbol = $2`, officialSector, symbol)
		if err != nil {
			fmt.Printf("  ❌ Failed to update %s: %v\n", symbol, err)
			continue
		}
		if tag.RowsAffected() > 0 {
			updatedCount++
			fmt.Printf("  ✅ %s -> %s\n", symbol, officialSector)
		}
	}

	fmt.Printf("🎉 Successfully updated %d stock sectors in database!\n", updatedCount)
}
