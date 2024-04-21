-- Drop existing tables
DROP TABLE IF EXISTS device CASCADE;
DROP TABLE IF EXISTS enrollment CASCADE;



-- Create tables
CREATE TABLE if not exists device (
  id SERIAL PRIMARY KEY,
  device_id VARCHAR(255) NOT NULL UNIQUE,
  name VARCHAR(255) NOT NULL,
  public_key TEXT NULL,
  last_seen TIMESTAMP
);

CREATE TABLE if not exists enrollment (
  id integer PRIMARY KEY references device(id) ON DELETE CASCADE,
  key VARCHAR(255) NOT NULL UNIQUE,
  device_settings pg_catalog.jsonb NOT NULL
);
