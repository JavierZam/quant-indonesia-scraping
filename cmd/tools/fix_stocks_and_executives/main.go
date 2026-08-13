package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"

	"github.com/JavierZam/quant-indonesia-scraping/pkg/sector"
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
	fmt.Println("🧹 Step 1: Cleaning up non-stock entities (BEI, OJK, IHSG, AMZN, etc.)...")

	blacklisted := []string{
		"AMZN", "INTC", "GS", "OJK", "BEI", "IDX", "IHSG", "KSEI", "BI", "PNM", "TMMIN", "RANS", "JECX", "BAIS", "KAI",
	}

	for _, sym := range blacklisted {
		_, _ = conn.Exec(ctx, "DELETE FROM news_stock_tags WHERE symbol = $1", sym)
		_, _ = conn.Exec(ctx, "DELETE FROM stock_prices WHERE symbol = $1", sym)
		_, _ = conn.Exec(ctx, "DELETE FROM stock_profiles WHERE symbol = $1", sym)
		_, _ = conn.Exec(ctx, "DELETE FROM executives WHERE symbol = $1", sym)
		_, _ = conn.Exec(ctx, "DELETE FROM stocks WHERE symbol = $1", sym)
		fmt.Printf("  🗑️ Deleted non-stock symbol: %s\n", sym)
	}

	fmt.Println("\n🏷️ Step 2: Fixing sectors & official names for all tracked stocks...")
	var updatedCount int
	for sym, officialSector := range sector.OfficialIDXMap {
		officialName := sector.GetOfficialCompanyName(sym, sym)

		query := `INSERT INTO stocks (symbol, company_name, sector, created_at, updated_at)
				  VALUES ($1, $2, $3, NOW(), NOW())
				  ON CONFLICT (symbol) DO UPDATE SET
					  company_name = EXCLUDED.company_name,
					  sector = EXCLUDED.sector,
					  updated_at = NOW()`

		_, err := conn.Exec(ctx, query, sym, officialName, officialSector)
		if err != nil {
			fmt.Printf("  ❌ Failed to update %s: %v\n", sym, err)
			continue
		}
		updatedCount++
		fmt.Printf("  ✅ %s -> %s [%s]\n", sym, officialName, officialSector)
	}
	fmt.Printf("🎉 Successfully updated %d stocks with official sectors & names!\n", updatedCount)

	fmt.Println("\n👔 Step 3: Populating detailed executive data for all major emiten...")
	executives := []ExecSeed{
		// Perbankan & Keuangan
		{"BBCA", "Jahja Setiaatmadja", "Presiden Direktur"},
		{"BBCA", "Armand Wahyudi Hartono", "Wakil Presiden Direktur"},
		{"BBCA", "Vera Eve Lim", "Direktur Keuangan"},
		{"BBCA", "Djohan Emir Setijoso", "Komisaris Utama"},
		{"BBRI", "Sunarso", "Direktur Utama"},
		{"BBRI", "Catur Budi Harto", "Wakil Direktur Utama"},
		{"BBRI", "Viviana Dyah Ayu Retno", "Direktur Keuangan"},
		{"BBRI", "Kartika Wirjoatmodjo", "Komisaris Utama"},
		{"BMRI", "Darmawan Junaidi", "Direktur Utama"},
		{"BMRI", "Alexandra Askandar", "Wakil Direktur Utama"},
		{"BMRI", "Sigit Prastowo", "Direktur Keuangan"},
		{"BMRI", "M. Chatib Basri", "Komisaris Utama"},
		{"BBNI", "Royke Tumilaar", "Direktur Utama"},
		{"BBNI", "Putrama Wahju Setyawan", "Wakil Direktur Utama"},
		{"BBNI", "Novita Widya Anggraini", "Direktur Keuangan"},
		{"BBNI", "Pradjoto", "Komisaris Utama"},
		{"BRIS", "Hery Gunardi", "Direktur Utama"},
		{"BRIS", "Bob Tyasika Ananta", "Wakil Direktur Utama"},
		{"BRIS", "Ade Cahyo Nugroho", "Direktur Keuangan"},
		{"BRIS", "Muliaman D. Hadad", "Komisaris Utama"},
		{"BBTN", "Nixon LP Napitupulu", "Direktur Utama"},
		{"BBTN", "Oni Febriarto Rahardjo", "Wakil Direktur Utama"},
		{"BBTN", "Nofry Rony Poetra", "Direktur Finance"},
		{"BBTN", "Chandra M. Hamzah", "Komisaris Utama"},
		{"ARTO", "Arief Harris Tandjung", "Direktur Utama"},
		{"ARTO", "Peterjan van Nieuwenhuizen", "Direktur Digital Banking"},
		{"ARTO", "Jerry Ng", "Komisaris Utama"},
		{"BNGA", "Lani Darmawan", "Presiden Direktur"},
		{"BNGA", "Lee Kai Kwong", "Direktur Finance & Strategy"},
		{"BNGA", "Didi Syafruddin Arifin", "Komisaris Utama"},
		{"MEGA", "Kostaman Thayib", "Direktur Utama"},
		{"MEGA", "Yuni Lastianto", "Direktur Risk Management"},
		{"BFIN", "Sutadi", "Direktur Utama"},
		{"ADMF", "I Made Sukada", "Direktur Utama"},

		// Telekomunikasi & Infrastruktur
		{"TLKM", "Ririek Adriansyah", "Direktur Utama"},
		{"TLKM", "Heri Supriadi", "Direktur Keuangan"},
		{"TLKM", "Bambang Brodjonegoro", "Komisaris Utama"},
		{"ISAT", "Vikram Sinha", "President Director & CEO"},
		{"ISAT", "Nicky Lee Chi Hung", "Director & CFO"},
		{"ISAT", "Halim Alamsyah", "Komisaris Utama"},
		{"EXCL", "Dian Siswarini", "President Director & CEO"},
		{"EXCL", "Feiruz Ikhwan", "Chief Financial Officer"},
		{"EXCL", "Shahril Tarmizi", "Komisaris Utama"},
		{"TOWR", "Amang Firdaus", "Direktur Utama"},
		{"JSMR", "Subakti Syamsurizall", "Direktur Utama"},
		{"JSMR", "Pramitha Wulansari", "Direktur Keuangan"},
		{"JSMR", "M. Zainal Fatah", "Komisaris Utama"},
		{"WIKA", "Agung Budi Waskito", "Direktur Utama"},
		{"WIKA", "Adityo Kusumo", "Direktur Keuangan"},
		{"WSKT", "Muhammad Hanugroho", "Direktur Utama"},
		{"WSKT", "Wiwi Suprihatno", "Direktur Keuangan"},
		{"ADHI", "Entus Asnawi Mukhson", "Direktur Utama"},
		{"ADHI", "Bambang Krisminarno", "Direktur Keuangan"},

		// Energi & Tambang
		{"ADRO", "Garibaldi Thohir", "Presiden Direktur & CEO"},
		{"ADRO", "Christian Ariano Rachmat", "Wakil Presiden Direktur"},
		{"ADRO", "Lie Luckman", "Direktur Finance"},
		{"ADRO", "Edwin Soeryadjaya", "Komisaris Utama"},
		{"AADI", "Julius Aslan", "Presiden Direktur"},
		{"AADI", "Garibaldi Thohir", "Komisaris Utama"},
		{"ADMR", "Christian Ariano Rachmat", "Presiden Direktur"},
		{"ADMR", "Heri Gunawan", "Direktur"},
		{"AKRA", "Haryanto Adikoesoemo", "Presiden Direktur"},
		{"AKRA", "Suresh Vembu", "Direktur Finance"},
		{"AKRA", "Soegiarto Adikoesoemo", "Komisaris Utama"},
		{"BUMI", "Adika Nuraga Bakrie", "Presiden Direktur"},
		{"BUMI", "Andrew C. Beckham", "Direktur Finance"},
		{"BUMI", "Anindya Novyan Bakrie", "Komisaris Utama"},
		{"BRPT", "Agus Salim Pangestu", "Presiden Direktur"},
		{"BRPT", "David Kosasih", "Direktur Finance"},
		{"BRPT", "Prajogo Pangestu", "Pendiri & Komisaris Utama"},
		{"BREN", "Hendra Soetjipto Tan", "Presiden Direktur"},
		{"BREN", "Merly", "Direktur Finance"},
		{"BREN", "Prajogo Pangestu", "Komisaris Utama"},
		{"TPIA", "Erwin Ciputra", "Presiden Direktur"},
		{"TPIA", "Baritono Pangestu", "Direktur"},
		{"TPIA", "Prajogo Pangestu", "Komisaris Utama"},
		{"ANTM", "Nico Kanter", "Direktur Utama"},
		{"ANTM", "Elisabeth RT Siahaan", "Direktur Keuangan"},
		{"INCO", "Febriany Eddy", "Presiden Direktur & CEO"},
		{"INCO", "Bernardus Irmanto", "Vice President Director"},
		{"PTBA", "Arsal Ismail", "Direktur Utama"},
		{"PTBA", "Farida Thamrin", "Direktur Keuangan & Manajemen Risiko"},
		{"PTRO", "Michael", "Presiden Direktur"},
		{"PTRO", "Kartika Hendrawan", "Direktur Finance"},
		{"PGAS", "Arief Setiawan Handoko", "Direktur Utama"},
		{"PGAS", "Fadjar Harianto Widodo", "Direktur Keuangan"},
		{"PGEO", "Julfi Hadi", "Direktur Utama"},
		{"PGEO", "Yurizki Rio", "Direktur Keuangan"},
		{"MEDC", "Hilmi Panigoro", "Direktur Utama"},
		{"MEDC", "Anthony R. Mathias", "Direktur Keuangan"},
		{"DSSA", "LKishore Kumar", "Presiden Direktur"},
		{"DSSA", "Franky Oesman Widjaja", "Komisaris Utama"},
		{"AMMN", "Alexander Ramlie", "Presiden Direktur / CEO"},
		{"AMMN", "Arief Widyawan Sidarto", "Direktur Keuangan"},

		// Teknologi & Ecommerce
		{"GOTO", "Patrick Sugito Walujo", "Direktur Utama / Group CEO"},
		{"GOTO", "Thomas Husted", "Vice President Director / COO"},
		{"GOTO", "Hans Patuwo", "Chief Operating Officer"},
		{"GOTO", "Agus DW Martowardojo", "Komisaris Utama"},
		{"BUKA", "Willix Halim", "Direktur Utama / CEO"},
		{"BUKA", "Teddy Nuryanto Oetomo", "Direktur"},
		{"EMTK", "Sutanto Hartono", "Direktur Utama / Managing Director"},
		{"EMTK", "Alvin W. Sariaatmadja", "Direktur"},
		{"BELI", "Kusumo Martanto", "CEO & Co-Founder"},

		// Konsumer & Farmasi
		{"ASII", "Djony Bunarto Tjondro", "Presiden Direktur"},
		{"ASII", "FXL Kesuma", "Direktur"},
		{"ASII", "Prijono Sugiarto", "Komisaris Utama"},
		{"AUTO", "Hamdhani Dzulkarnaen Salim", "Presiden Direktur"},
		{"KLBF", "Vidjongtius", "Presiden Direktur"},
		{"KLBF", "Bernadus Karmin Winata", "Direktur Finance"},
		{"SIDO", "David Hidayat", "Direktur Utama"},
		{"SIDO", "Leonard", "Direktur Finance"},
		{"KAEF", "Djagad Prakoso Dwialam", "Direktur Utama"},
		{"KAEF", "Lina Jennie", "Direktur Keuangan"},
		{"HEAL", "Arfan Awaloeddin", "Direktur Utama"},
		{"AVIA", "Wijono Tanoko", "Presiden Direktur"},
		{"AVIA", "Hermanto Tanoko", "Komisaris Utama"},
		{"ICBP", "Anthoni Salim", "Direktur Utama / CEO"},
		{"INDF", "Anthoni Salim", "Direktur Utama / CEO"},
		{"UNVR", "Benjie Yap", "Presiden Direktur"},
		{"UNVR", "Vivek Agarwal", "Direktur Finance"},
		{"MYOR", "Andre Sukendra Atmadja", "Presiden Direktur"},
		{"GGRM", "Susilo Wonowidjojo", "Presiden Direktur"},
		{"HMSP", "Ivan Cahyadi", "Presiden Direktur"},
		{"AMRT", "Anggara Hans Prawira", "Presiden Direktur"},
		{"MAPI", "V.P. Sharma", "Group CEO & Vice President Director"},
		{"CPIN", "Thomas Effendy", "Presiden Direktur"},
		{"JPFA", "Handojo Santosa", "Direktur Utama / CEO"},
		{"CMRY", "Fareel Sutantio", "Direktur Utama"},

		// Properti & Industrials
		{"BSDE", "Franciscus Xaverius Ridwan Darmali", "Presiden Direktur"},
		{"BSDE", "Hermawan Wijaya", "Direktur Finance"},
		{"CTRA", "Candra Ciputra", "Direktur Utama"},
		{"SMRA", "Adrianto P. Adhi", "Direktur Utama"},
		{"UNTR", "Franskesjan", "Presiden Direktur"},
		{"UNTR", "Djony Bunarto Tjondro", "Komisaris Utama"},
	}

	var execCount int
	for _, exec := range executives {
		query := `INSERT INTO executives (symbol, name, title, created_at, updated_at)
				  VALUES ($1, $2, $3, NOW(), NOW())
				  ON CONFLICT (symbol, name) DO UPDATE SET title = EXCLUDED.title, updated_at = NOW()`
		_, err := conn.Exec(ctx, query, exec.Symbol, exec.Name, exec.Title)
		if err != nil {
			fmt.Printf("  ❌ Failed to insert executive %s (%s): %v\n", exec.Name, exec.Symbol, err)
			continue
		}
		execCount++
	}

	fmt.Printf("🎉 Successfully populated %d executives across all tracked stocks!\n", execCount)
}
