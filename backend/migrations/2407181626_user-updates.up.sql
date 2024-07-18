DELETE FROM "user";

ALTER TABLE "user" ADD COLUMN display_name VARCHAR(255) NOT NULL DEFAULT '';
ALTER TABLE "user" ADD COLUMN idp_user_id VARCHAR(255) NOT NULL UNIQUE DEFAULT '';

ALTER TABLE "user" DROP CONSTRAINT IF EXISTS user_email_key;
DROP INDEX IF EXISTS user_email_key;


CREATE TABLE tenant_invite (
    id SERIAL PRIMARY KEY,
    tenant_id integer references tenant(id) ON DELETE CASCADE,
    invite_id VARCHAR(255) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL,
    role tenant_role NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

DELETE FROM user_tenant;
ALTER TABLE user_tenant ADD COLUMN member_id VARCHAR(255) NOT NULL UNIQUE DEFAULT '';
