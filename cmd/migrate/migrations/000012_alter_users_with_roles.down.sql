-- Drop the role_id column if it already exists
ALTER TABLE users
DROP COLUMN IF EXISTS role_id;
