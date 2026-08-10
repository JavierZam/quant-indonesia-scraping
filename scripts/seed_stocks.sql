-- Seed data: Top Indonesian (IDX) stocks by market cap
-- LQ45 + major companies

INSERT INTO stocks (symbol, company_name, sector) VALUES
-- Banking
('BBCA', 'Bank Central Asia Tbk', 'Banking'),
('BBRI', 'Bank Rakyat Indonesia Tbk', 'Banking'),
('BMRI', 'Bank Mandiri Tbk', 'Banking'),
('BBNI', 'Bank Negara Indonesia Tbk', 'Banking'),
('BRIS', 'Bank Syariah Indonesia Tbk', 'Banking'),

-- Telco
('TLKM', 'Telkom Indonesia Tbk', 'Telecommunications'),
('EXCL', 'XL Axiata Tbk', 'Telecommunications'),
('ISAT', 'Indosat Ooredoo Hutchison Tbk', 'Telecommunications'),

-- Consumer Goods
('UNVR', 'Unilever Indonesia Tbk', 'Consumer Goods'),
('ICBP', 'Indofood CBP Sukses Makmur Tbk', 'Consumer Goods'),
('INDF', 'Indofood Sukses Makmur Tbk', 'Consumer Goods'),
('MYOR', 'Mayora Indah Tbk', 'Consumer Goods'),

-- Mining & Resources
('ADRO', 'Adaro Energy Indonesia Tbk', 'Mining'),
('ANTM', 'Aneka Tambang Tbk', 'Mining'),
('INCO', 'Vale Indonesia Tbk', 'Mining'),
('PTBA', 'Bukit Asam Tbk', 'Mining'),
('MDKA', 'Merdeka Copper Gold Tbk', 'Mining'),

-- Automotive
('ASII', 'Astra International Tbk', 'Automotive'),
('AUTO', 'Astra Otoparts Tbk', 'Automotive'),

-- Property & Construction
('BSDE', 'Bumi Serpong Damai Tbk', 'Property'),
('CTRA', 'Ciputra Development Tbk', 'Property'),
('SMRA', 'Summarecon Agung Tbk', 'Property'),

-- Infrastructure
('JSMR', 'Jasa Marga Tbk', 'Infrastructure'),
('WIKA', 'Wijaya Karya Tbk', 'Infrastructure'),

-- Healthcare
('KLBF', 'Kalbe Farma Tbk', 'Healthcare'),
('SIDO', 'Industri Jamu dan Farmasi Sido Muncul Tbk', 'Healthcare'),

-- Technology
('GOTO', 'GoTo Gojek Tokopedia Tbk', 'Technology'),
('BUKA', 'Bukalapak.com Tbk', 'Technology'),
('EMTK', 'Elang Mahkota Teknologi Tbk', 'Technology')

ON CONFLICT (symbol) DO UPDATE SET
  company_name = EXCLUDED.company_name,
  sector = EXCLUDED.sector,
  updated_at = NOW();
