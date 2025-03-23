CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(255) NULL UNIQUE,
    chat_id INT UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    city VARCHAR(255),
    timezone VARCHAR(3),
    mailing_time TIME
);

CREATE INDEX idx_chat_id ON users (chat_id);
CREATE INDEX idx_mailing_time ON users (mailing_time);

