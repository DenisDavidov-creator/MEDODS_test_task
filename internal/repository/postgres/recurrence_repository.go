package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type RecurrenceRepository struct {
	pool *pgxpool.Pool
}

func NewRecurrenceRepository(pool *pgxpool.Pool) *RecurrenceRepository {
	return &RecurrenceRepository{pool: pool}
}

func (r *RecurrenceRepository) Create(ctx context.Context)
