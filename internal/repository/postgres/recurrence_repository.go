package postgres

import (
	"context"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RecurrenceRepository struct {
	pool *pgxpool.Pool
}

func NewRecurrenceRepository(pool *pgxpool.Pool) *RecurrenceRepository {
	return &RecurrenceRepository{pool: pool}
}

func (r *RecurrenceRepository) Create(ctx context.Context, recurrence *taskdomain.Recurrence) (*taskdomain.Recurrence, error) {
	return nil, nil
}
func (r *RecurrenceRepository) GetByTaskID(ctx context.Context, taskID int64) (*taskdomain.Recurrence, error) {
	return nil, nil
}
func (r *RecurrenceRepository) Update(ctx context.Context, recurrence *taskdomain.Recurrence) (*taskdomain.Recurrence, error) {
	return nil, nil
}
func (r *RecurrenceRepository) Delete(ctx context.Context, id int64) error {
	return nil
}
func (r *RecurrenceRepository) ListDue(ctx context.Context, now time.Time) ([]taskdomain.Recurrence, error) {
	return nil, nil
}
