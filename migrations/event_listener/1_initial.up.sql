CREATE TABLE subscriptions (
    token_id  BIGINT         NOT NULL PRIMARY KEY,
    mint_id   VARBINARY(16)  NOT NULL,
    timeout   BIGINT         NOT NULL
);
