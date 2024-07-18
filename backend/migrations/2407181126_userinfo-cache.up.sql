-- This migration creates the userinfo_cache table, which is used to store the userinfo response from the userinfo endpoint.
CREATE TABLE userinfo_cache (
    token_hash VARCHAR(255) PRIMARY KEY,
    userinfo TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
