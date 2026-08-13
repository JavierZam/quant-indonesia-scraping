package sector

import "strings"

// OfficialIDXMap maps stock ticker symbols to their official IDX sector categories.
var OfficialIDXMap = map[string]string{
	// Energy / Pertambangan & Minyak Gas
	"BUMI": "Energy",
	"DSSA": "Energy",
	"ADRO": "Energy",
	"PTBA": "Energy",
	"ITMG": "Energy",
	"HRUM": "Energy",
	"MEDC": "Energy",
	"PGAS": "Energy",
	"AKRA": "Energy",
	"ENRG": "Energy",
	"PTRO": "Energy",
	"RAJA": "Energy",
	"RATU": "Energy",

	// Basic Materials / Barang Baku & Tambang Mineral
	"BRPT": "Basic Materials",
	"TPIA": "Basic Materials",
	"BRMS": "Basic Materials",
	"ANTM": "Basic Materials",
	"INCO": "Basic Materials",
	"TINS": "Basic Materials",
	"MDKA": "Basic Materials",
	"AMMN": "Basic Materials",
	"MBMA": "Basic Materials",
	"NCKL": "Basic Materials",
	"INKP": "Basic Materials",
	"TKIM": "Basic Materials",
	"IMPC": "Basic Materials",

	// Consumer Goods / Non-Cyclicals (Konsumer Primer, Makanan, Rokok)
	"GGRM": "Consumer Goods",
	"HMSP": "Consumer Goods",
	"INDF": "Consumer Goods",
	"ICBP": "Consumer Goods",
	"UNVR": "Consumer Goods",
	"MYOR": "Consumer Goods",
	"CPIN": "Consumer Goods",
	"JPFA": "Consumer Goods",

	// Consumer Cyclicals / Siklikal (Otomotif, Ritel, Kendaraan)
	"ASII": "Consumer Cyclicals",
	"AUTO": "Consumer Cyclicals",
	"VKTR": "Consumer Cyclicals",
	"MAPI": "Consumer Cyclicals",
	"ACES": "Consumer Cyclicals",
	"RALS": "Consumer Cyclicals",

	// Financial Services / Keuangan & Perbankan
	"BBCA": "Financial Services",
	"BBRI": "Financial Services",
	"BMRI": "Financial Services",
	"BBNI": "Financial Services",
	"BRIS": "Financial Services",
	"BBTN": "Financial Services",
	"ARTO": "Financial Services",
	"BNGA": "Financial Services",
	"BBSI": "Financial Services",
	"MEGA": "Financial Services",
	"SMMA": "Financial Services",

	// Infrastructure / Telekomunikasi, Menara, Tol & Konstruksi
	"TLKM": "Infrastructure",
	"ISAT": "Infrastructure",
	"EXCL": "Infrastructure",
	"TOWR": "Infrastructure",
	"TBIG": "Infrastructure",
	"JSMR": "Infrastructure",
	"WIKA": "Infrastructure",
	"PTPP": "Infrastructure",
	"ADHI": "Infrastructure",
	"DCII": "Infrastructure",
	"RLCO": "Infrastructure",
	"EAST": "Infrastructure",
	"KOTA": "Infrastructure",

	// Technology
	"GOTO": "Technology",
	"BUKA": "Technology",
	"EMTK": "Technology",

	// Healthcare
	"KLBF": "Healthcare",
	"SIDO": "Healthcare",
	"MIKA": "Healthcare",
	"HEAL": "Healthcare",
	"SILO": "Healthcare",

	// Real Estate & Property
	"BSDE": "Real Estate",
	"CTRA": "Real Estate",
	"SMRA": "Real Estate",
	"PWON": "Real Estate",
	"SINI": "Real Estate",
}

// NormalizeSector returns the official IDX sector if known, or normalizes the provided tag.
func NormalizeSector(symbol string, tagSector string) string {
	sym := strings.ToUpper(strings.TrimSpace(symbol))
	if official, ok := OfficialIDXMap[sym]; ok {
		return official
	}

	tag := strings.TrimSpace(tagSector)
	if tag == "" || strings.EqualFold(tag, "Finansial") || strings.EqualFold(tag, "Financial Services") || strings.EqualFold(tag, "Financial") {
		// If tag is generic financial and emiten is not in known map, return blank or default
		return tag
	}

	return tag
}
