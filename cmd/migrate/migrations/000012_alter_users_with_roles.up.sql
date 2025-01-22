BEGIN;

-- Add the new column without a default value
ALTER TABLE users
ADD COLUMN role_id BIGINT REFERENCES roles (id);

-- Update the role_id column based on the 'user' role
UPDATE users
SET
    role_id = (
        SELECT
            id
        FROM
            roles
        WHERE
            name = 'user'
    );

-- Ensure all rows have a valid role_id
-- If no 'user' role exists, this will fail
ALTER TABLE users
ALTER COLUMN role_id
SET
    NOT NULL;

COMMIT;
