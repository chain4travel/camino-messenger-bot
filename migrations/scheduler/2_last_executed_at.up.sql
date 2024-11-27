ALTER TABLE jobs
DROP execute_at;

ALTER TABLE jobs
ADD last_executed_at BIGINT NOT NULL DEFAULT 0;
