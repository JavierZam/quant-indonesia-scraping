package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
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

	// 1. Run Migrations
	migrationFiles := []string{
		"migrations/000001_create_stocks_table.up.sql",
		"migrations/000002_create_executives_table.up.sql",
		"migrations/000003_create_news_articles_table.up.sql",
		"migrations/000004_create_news_stock_tags_table.up.sql",
		"migrations/000005_create_stock_prices_table.up.sql",
		"migrations/000006_add_unique_index_to_executives.up.sql",
	}

	fmt.Println("🚀 Executing migrations...")
	for _, file := range migrationFiles {
		content, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("❌ Failed to read migration file %s: %v\n", file, err)
			os.Exit(1)
		}

		_, err = conn.Exec(ctx, string(content))
		if err != nil {
			fmt.Printf("❌ Error executing %s: %v\n", file, err)
			os.Exit(1)
		}
		fmt.Printf("  ✅ Applied %s\n", filepath.Base(file))
	}

	// 2. Run Seed
	seedFile := "scripts/seed_stocks.sql"
	fmt.Println("🌱 Seeding stock data...")
	seedContent, err := os.ReadFile(seedFile)
	if err != nil {
		fmt.Printf("❌ Failed to read seed file %s: %v\n", seedFile, err)
		os.Exit(1)
	}

	_, err = conn.Exec(ctx, string(seedContent))
	if err != nil {
		fmt.Printf("❌ Error executing seed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("  ✅ Stocks seeded successfully!")

	// Count stocks
	var count int
	_ = conn.QueryRow(ctx, "SELECT count(*) FROM stocks").Scan(&count)
	fmt.Printf("🎉 Database setup complete! Total stocks in database: %d\n", count)
}
