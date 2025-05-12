CREATE TABLE jobs (
    name        VARCHAR(100)  NOT NULL PRIMARY KEY,
    execute_at  BIGINT        NOT NULL,
    period      BIGINT        NOT NULL
);
