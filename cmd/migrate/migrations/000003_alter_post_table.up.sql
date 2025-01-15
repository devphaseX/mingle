ALTER TABLE posts ADD CONSTRAINT fk_user_post FOREIGN KEY (user_id) REFERENCES users (id)
