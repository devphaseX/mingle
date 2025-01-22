-- Create the roles table if it does not exist
CREATE TABLE IF NOT EXISTS roles (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    level INT NOT NULL DEFAULT 0,
    description TEXT
);

-- Insert roles if they do not already exist
INSERT INTO
    roles (name, description, level)
VALUES
    ('user', 'A user can create posts and comments', 1),
    (
        'moderator',
        'A moderator can update posts and comments',
        2
    ),
    (
        'admin',
        'An Admin can update and delete posts',
        3
    ) -- Added level for admin
    ON CONFLICT (name) DO NOTHING;

-- Prevents duplicate inserts
