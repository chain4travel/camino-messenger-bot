CREATE TABLE tokens (
    token_id      TEXT  NOT NULL PRIMARY KEY,
    bought       BOOLEAN       NOT NULL,
    expired      BOOLEAN       NOT NULL,
    created_at  BIGINT        NOT NULL,
    expires_at  BIGINT        NOT NULL
);
