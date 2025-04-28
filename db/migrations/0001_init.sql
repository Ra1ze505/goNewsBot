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

CREATE TABLE rates (
    date TIMESTAMP UNIQUE NOT NULL,
    data JSONB NOT NULL
);

CREATE INDEX idx_date ON rates (date);

CREATE TABLE IF NOT EXISTS messages (
    id SERIAL PRIMARY KEY,
    channel_id BIGINT NOT NULL,
    message_id BIGINT NOT NULL,
    channel_username VARCHAR(255) NOT NULL,
    message_text TEXT,
    message_date TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(channel_id, message_id)
); 