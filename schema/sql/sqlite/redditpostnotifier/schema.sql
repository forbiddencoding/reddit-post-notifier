BEGIN;

CREATE TABLE IF NOT EXISTS configuration
(
    id              BIGINT    NOT NULL,
    owner_id        BIGINT    NOT NULL DEFAULT 0, /* owner_id could be used for a multi-user application */
    keyword         TEXT      NOT NULL,
    schedule        TEXT      NOT NULL DEFAULT '0 0 * * *',
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS subreddit_configuration
(
    id                 BIGINT    NOT NULL,
    configuration_id   BIGINT    NOT NULL,
    subreddit          TEXT      NOT NULL,
    include_nsfw       BOOLEAN   NOT NULL DEFAULT FALSE,
    sort               TEXT      NOT NULL DEFAULT 'new',
    restrict_subreddit BOOLEAN   NOT NULL DEFAULT TRUE,
    created_at         TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_updated_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    FOREIGN KEY (configuration_id) REFERENCES configuration (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_subreddit_configuration_configuration_id
    ON subreddit_configuration (configuration_id);

CREATE TABLE IF NOT EXISTS subreddit_configuration_state
(
    subreddit_configuration_id BIGINT UNIQUE NOT NULL,
    last_post                  TEXT          NOT NULL,
    last_updated_at            TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (subreddit_configuration_id) REFERENCES subreddit_configuration (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS recipients
(
    id               BIGINT    NOT NULL,
    configuration_id BIGINT    NOT NULL,
    address          TEXT      NOT NULL,
    created_at       TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    FOREIGN KEY (configuration_id) REFERENCES configuration (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_recipients_configuration_id
    ON recipients (configuration_id);

COMMIT;