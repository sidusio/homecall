CREATE TABLE if not exists enrollment (
    id SERIAL PRIMARY KEY,
    key VARCHAR(255) NOT NULL UNIQUE,
    device_settings pg_catalog.jsonb NOT NULL,
    device_name VARCHAR(255) NOT NULL
);

CREATE TABLE if not exists device (
    id SERIAL PRIMARY KEY,
    enrollment_id INTEGER NOT NULL REFERENCES enrollment(id) ON DELETE CASCADE,
    device_id VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    public_key VARCHAR(255) NOT NULL
);
