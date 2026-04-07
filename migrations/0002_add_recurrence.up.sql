CREATE TABLE IF NOT EXISTS recurrence (
    id BIGSERIAL PRIMARY KEY,
    task_id BIGINT NOT NULL,
    type TEXT NOT NULL,
    interval INT,
    day_of_month INT,
    even_odd TEXT,
    next_run_at TIMESTAMPTZ,
    CONSTRAINT fk_recurrence_task FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
);

CREATE INDEX idx_recurrence_next_run_at ON recurrence (next_run_at) WHERE next_run_at IS NOT NULL;