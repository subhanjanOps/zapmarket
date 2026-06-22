-- Seed the 25 currencies from the seller-ui prototype list.
-- JPY, KRW, IDR are zero-decimal (decimals=0); all others use 2.
INSERT INTO currencies (code, name, flag, decimals, enabled) VALUES
  ('USD', 'US Dollar',           '🇺🇸', 2, true),
  ('EUR', 'Euro',                '🇪🇺', 2, true),
  ('GBP', 'British Pound',       '🇬🇧', 2, true),
  ('JPY', 'Japanese Yen',        '🇯🇵', 0, true),
  ('CAD', 'Canadian Dollar',     '🇨🇦', 2, true),
  ('AUD', 'Australian Dollar',   '🇦🇺', 2, true),
  ('CHF', 'Swiss Franc',         '🇨🇭', 2, true),
  ('CNY', 'Chinese Yuan',        '🇨🇳', 2, true),
  ('INR', 'Indian Rupee',        '🇮🇳', 2, true),
  ('BRL', 'Brazilian Real',      '🇧🇷', 2, true),
  ('MXN', 'Mexican Peso',        '🇲🇽', 2, true),
  ('SGD', 'Singapore Dollar',    '🇸🇬', 2, true),
  ('HKD', 'Hong Kong Dollar',    '🇭🇰', 2, true),
  ('NOK', 'Norwegian Krone',     '🇳🇴', 2, true),
  ('SEK', 'Swedish Krona',       '🇸🇪', 2, true),
  ('DKK', 'Danish Krone',        '🇩🇰', 2, true),
  ('NZD', 'New Zealand Dollar',  '🇳🇿', 2, true),
  ('ZAR', 'South African Rand',  '🇿🇦', 2, true),
  ('AED', 'UAE Dirham',          '🇦🇪', 2, true),
  ('SAR', 'Saudi Riyal',         '🇸🇦', 2, true),
  ('KRW', 'South Korean Won',    '🇰🇷', 0, true),
  ('THB', 'Thai Baht',           '🇹🇭', 2, true),
  ('IDR', 'Indonesian Rupiah',   '🇮🇩', 0, true),
  ('MYR', 'Malaysian Ringgit',   '🇲🇾', 2, true),
  ('PHP', 'Philippine Peso',     '🇵🇭', 2, true)
ON CONFLICT (code) DO NOTHING;
