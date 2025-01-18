CREATE TABLE IF NOT EXISTS user_invitations (
    token bytea PRIMARY KEY,
    user_id BIGINT NOT NULL,
    expiry TIMESTAMP
    WITH
        TIME ZONE NOT NULL
)
