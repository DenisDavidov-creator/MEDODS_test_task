CREATE TABLE IF NOT EXISTS recurrence_dates (
    id BIGSERIAL PRIMARY KEY,
    recurrence_id BIGINT NOT NULL,
    date DATE NOT NULL,
    CONSTRAINT fk_recurrence_dates FOREIGN KEY (recurrence_id) REFERENCES recurrence(id) ON DELETE CASCADE
);