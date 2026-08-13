package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

type ExecSeed struct {
	Symbol string
	Name   string
	Title  string
}

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
	fmt.Println("👔 Populating executives table with key corporate figures & directors...")

	executives := []ExecSeed{
		{"BBCA", "Jahja Setiaatmadja", "Presiden Direktur"},
		{"BBCA", "Armand Wahyudi Hartono", "Wakil Presiden Direktur"},
		{"BBRI", "Sunarso", "Direktur Utama"},
		{"BBRI", "Catur Budi Harto", "Wakil Direktur Utama"},
		{"BMRI", "Darmawan Junaidi", "Direktur Utama"},
		{"BMRI", "Alexandra Askandar", "Wakil Direktur Utama"},
		{"BBNI", "Royke Tumilaar", "Direktur Utama"},
		{"BBNI", "Destry Damayanti", "Senior Executive / Ex-Commissioner"},
		{"BRIS", "Hery Gunardi", "Direktur Utama"},
		{"JSMR", "Rivan A. Purwantono", "Direktur Utama"},
		{"TLKM", "Ririek Adriansyah", "Direktur Utama"},
		{"ISAT", "Vikram Sinha", "President Director & CEO"},
		{"EXCL", "Dian Siswarini", "President Director & CEO"},
		{"ASII", "Djony Bunarto Tjondro", "Presiden Direktur"},
		{"ADRO", "Garibaldi Thohir", "Presiden Direktur & CEO"},
		{"BRPT", "Prajogo Pangestu", "Pendiri & Komisaris Utama"},
		{"TPIA", "Erri Febrianto", "Presiden Direktur"},
		{"BUMI", "Adika Nuraga Bakrie", "Presiden Direktur"},
		{"BUMI", "Anindya Novyan Bakrie", "Komisaris Utama"},
		{"DSSA", "Franky Oesman Widjaja", "Komisaris Utama"},
		{"DSSA", "LKishore Kumar", "Presiden Direktur"},
		{"ANTM", "Nico Kanter", "Direktur Utama"},
		{"INCO", "Febriany Eddy", "Presiden Direktur & CEO"},
		{"PTBA", "Arsal Ismail", "Direktur Utama"},
		{"GOTO", "Patrick Walujo", "Direktur Utama / Group CEO"},
		{"BUKA", "Willix Halim", "Direktur Utama / CEO"},
		{"EMTK", "Sutanto Hartono", "Direktur Utama"},
		{"KLBF", "Vidjongtius", "Presiden Direktur"},
		{"SIDO", "David Hidayat", "Direktur Utama"},
		{"BSDE", "Michael JP Widjaja", "Presiden Direktur"},
		{"CTRA", "Candra Ciputra", "Direktur Utama"},
	}

	var insertedCount int
	for _, exec := range executives {
		query := `INSERT INTO executives (symbol, name, title, created_at, updated_at)
				  VALUES ($1, $2, $3, NOW(), NOW())
				  ON CONFLICT (symbol, name) DO UPDATE SET title = EXCLUDED.title, updated_at = NOW()`
		_, err := conn.Exec(ctx, query, exec.Symbol, exec.Name, exec.Title)
		if err != nil {
			fmt.Printf("  ❌ Failed to insert %s (%s): %v\n", exec.Name, exec.Symbol, err)
			continue
		}
		insertedCount++
		fmt.Printf("  ✅ %s -> %s (%s)\n", exec.Symbol, exec.Name, exec.Title)
	}

	fmt.Printf("🎉 Successfully populated %d executives in database!\n", insertedCount)
}
