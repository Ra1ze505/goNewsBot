-- db/migrations/0002_storylines.sql
-- Storyline Tracking / TDT: долгоживущие сюжетные линии на канал и их дневные наблюдения.
-- Применяется вручную, как 0001 (см. AGENTS.md). Требует установленного пакета pgvector.

CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS storylines (
    id           SERIAL PRIMARY KEY,
    channel_id   BIGINT NOT NULL,
    title        TEXT   NOT NULL,                 -- каноничный заголовок (стабильный)
    state        TEXT   NOT NULL,                 -- "сводка обстановки": текущее состояние сюжета
    category     TEXT,                            -- рубрика: военное / происшествия / экономика / ...
    status       TEXT   NOT NULL DEFAULT 'active',-- active | dormant | closed
    importance   INT    NOT NULL DEFAULT 1,       -- актуальная важность 1..5
    embedding    vector(256),                     -- doc-эмбеддинг по title+state (Yandex text-search-doc)
    first_seen   DATE   NOT NULL,
    last_seen    DATE   NOT NULL,
    created_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_storylines_channel_status   ON storylines(channel_id, status);
CREATE INDEX IF NOT EXISTS idx_storylines_channel_lastseen ON storylines(channel_id, last_seen);

-- ANN-индекс для косинусного поиска. ivfflat требует ANALYZE/наполнения;
-- для малых объёмов можно начать без индекса (точный перебор) и добавить позже.
CREATE INDEX IF NOT EXISTS idx_storylines_embedding
    ON storylines USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

CREATE TABLE IF NOT EXISTS storyline_observations (
    id            SERIAL PRIMARY KEY,
    storyline_id  INT  NOT NULL REFERENCES storylines(id) ON DELETE CASCADE,
    channel_id    BIGINT NOT NULL,
    obs_date      DATE NOT NULL,
    message_count INT  NOT NULL DEFAULT 0,    -- сколько сообщений легло в сюжет в этот день
    importance    INT  NOT NULL DEFAULT 1,    -- важность сюжета в этот день
    change_type   TEXT NOT NULL,              -- new|escalation|ongoing|deescalation|recurring_noise
    delta_summary TEXT,                       -- что именно нового в этот день
    source_message_ids BIGINT[],              -- реальные message_id из messages
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (storyline_id, obs_date)           -- идемпотентность при перегенерации/бэкфилле
);

CREATE INDEX IF NOT EXISTS idx_obs_channel_date ON storyline_observations(channel_id, obs_date);
CREATE INDEX IF NOT EXISTS idx_obs_storyline    ON storyline_observations(storyline_id);
