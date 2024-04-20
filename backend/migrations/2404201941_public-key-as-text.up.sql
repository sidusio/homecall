
-- Change the type of public_key from VARCHAR(255) to TEXT
ALTER TABLE if exists device ALTER COLUMN public_key TYPE TEXT;
