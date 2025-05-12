ALTER TABLE jobs
DROP last_executed_at;

ALTER TABLE jobs
ADD execute_at BIGINT NOT NULL;