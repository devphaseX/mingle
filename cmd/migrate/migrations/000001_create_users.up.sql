CREATE TABLE IF NOT EXISTS users (
    id bigserial PRIMARY KEY,
    first_name varchar(255) NOT NULL,
    last_name varchar(255) NOT NULL,
    email citext UNIQUE NOT NULL,
    username varchar(255) NOT NULL,
    password_hash bytea NOT NULL,
    created_at timestamp(0)
    with
        time zone NOT NULL DEFAULT NOW ()
)
