-- Drop existing tables
DROP TABLE IF EXISTS device CASCADE;
DROP TABLE IF EXISTS enrollment CASCADE;



-- Create tables
CREATE TABLE if not exists tenant (
  id SERIAL PRIMARY KEY,
  tenant_id VARCHAR(255) NOT NULL UNIQUE,
  name VARCHAR(255) NOT NULL,
  max_devices integer NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE if not exists device (
  id SERIAL PRIMARY KEY,
  tenant_id integer references tenant(id) ON DELETE CASCADE,
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

CREATE TABLE "user" (
  id SERIAL PRIMARY KEY,
  email VARCHAR(255) NOT NULL UNIQUE
);

-- Role enum
CREATE TYPE tenant_role AS ENUM ('admin', 'user');

CREATE TABLE user_tenant (
  user_id integer references "user"(id) ON DELETE CASCADE,
  tenant_id integer references tenant(id) ON DELETE CASCADE,
  role tenant_role NOT NULL,
  PRIMARY KEY (user_id, tenant_id)
);
