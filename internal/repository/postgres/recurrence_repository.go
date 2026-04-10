package postgres

import (
	"context"
	"errors"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RecurrenceRepository struct {
	pool *pgxpool.Pool
}

func NewRecurrenceRepository(pool *pgxpool.Pool) *RecurrenceRepository {
	return &RecurrenceRepository{pool: pool}
}

func (r *RecurrenceRepository) Create(ctx context.Context, recurrence *taskdomain.Recurrence) (*taskdomain.Recurrence, error) {

	const query = `
		INSERT INTO recurrence (task_id, type, interval, day_of_month, even_odd, next_run_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, task_id, type, interval, day_of_month, even_odd, next_run_at
	`
	row := getQuerier(ctx, r.pool).QueryRow(ctx, query, recurrence.TaskID, recurrence.Type, recurrence.Interval, recurrence.DayOfMonth, recurrence.EvenOdd, recurrence.NextRunAt)
	created, err := scanRecurrence(row)
	if err != nil {
		return nil, err
	}
	if recurrence.Dates != nil {
		const queryDates = `
			INSERT INTO recurrence_dates (recurrence_id, date)
			VALUES ($1, $2)
		`
		batch := &pgx.Batch{}
		for _, date := range recurrence.Dates {
			batch.Queue(queryDates, created.ID, date)
		}
		batchResult := getQuerier(ctx, r.pool).SendBatch(ctx, batch)
		if err := batchResult.Close(); err != nil {
			return nil, err
		}
		created.Dates = recurrence.Dates
	}

	return created, nil
}

func (r *RecurrenceRepository) GetByTaskID(ctx context.Context, taskID int64) (*taskdomain.Recurrence, error) {
	const query = `
		SELECT id, task_id, type, interval, day_of_month, even_odd, next_run_at
		FROM recurrence
		WHERE task_id = $1
	`
	row := getQuerier(ctx, r.pool).QueryRow(ctx, query, taskID)
	found, err := scanRecurrence(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, taskdomain.ErrNotFound
		}
		return nil, err
	}
	if found.Type == taskdomain.TypeSpecificDates {
		const query = `
			SELECT date
			FROM recurrence_dates
			WHERE recurrence_id = $1 
		`
		rows, err := getQuerier(ctx, r.pool).Query(ctx, query, found.ID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var date time.Time
			if err := rows.Scan(&date); err != nil {
				return nil, err
			}
			found.Dates = append(found.Dates, date)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}
	return found, nil
}

func (r *RecurrenceRepository) Update(ctx context.Context, recurrence *taskdomain.Recurrence) (*taskdomain.Recurrence, error) {
	const query = `
			UPDATE recurrence
			SET type = $1,
				interval = $2,
				day_of_month = $3, 
				even_odd = $4,
				next_run_at = $5
			WHERE id = $6
			RETURNING id, task_id, type, interval, day_of_month, even_odd, next_run_at
		`

	row := getQuerier(ctx, r.pool).QueryRow(ctx, query, recurrence.Type, recurrence.Interval, recurrence.DayOfMonth, recurrence.EvenOdd, recurrence.NextRunAt, recurrence.ID)
	updatet, err := scanRecurrence(row)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, taskdomain.ErrNotFound
		}
		return nil, err
	}

	if recurrence.Dates != nil {
		_, err := getQuerier(ctx, r.pool).Exec(ctx, `DELETE FROM recurrence_dates WHERE recurrence_id = $1`, updatet.ID)
		if err != nil {
			return nil, err
		}
		const queryDates = `
			INSERT INTO recurrence_dates (recurrence_id, date)
			VALUES ($1, $2)
		`
		batch := &pgx.Batch{}
		for _, date := range recurrence.Dates {
			batch.Queue(queryDates, updatet.ID, date)
		}
		batchResult := getQuerier(ctx, r.pool).SendBatch(ctx, batch)
		if err := batchResult.Close(); err != nil {
			return nil, err
		}
		updatet.Dates = recurrence.Dates
	}

	return updatet, nil
}

func (r *RecurrenceRepository) Delete(ctx context.Context, id int64) error {
	const query = `
		DELETE FROM recurrence WHERE id = $1
		`
	_, err := getQuerier(ctx, r.pool).Exec(ctx, query, id)
	if err != nil {
		return err
	}
	return nil
}

func (r *RecurrenceRepository) ListDue(ctx context.Context, now time.Time) ([]taskdomain.Recurrence, error) {
	const query = `
		SELECT id, task_id, type, interval, day_of_month, even_odd, next_run_at
		FROM recurrence
		WHERE next_run_at <= $1
		`
	rows, err := getQuerier(ctx, r.pool).Query(ctx, query, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recurrences []taskdomain.Recurrence
	for rows.Next() {
		rec, err := scanRecurrence(rows)
		if err != nil {
			return nil, err
		}
		recurrences = append(recurrences, *rec)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return recurrences, nil
}

func (r *RecurrenceRepository) ListByTaskIDs(ctx context.Context, taskIDs []int64) ([]taskdomain.Recurrence, error) {
	const query = `
		SELECT r.id, r.task_id, r.type, r.interval, r.day_of_month, r.even_odd, r.next_run_at, rd.date
		FROM recurrence r
		LEFT JOIN recurrence_dates rd ON rd.recurrence_id = r.id
		WHERE task_id = ANY($1) 
	`
	rows, err := getQuerier(ctx, r.pool).Query(ctx, query, taskIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	recurrenceMap := map[int64]*taskdomain.Recurrence{}
	for rows.Next() {
		var rec taskdomain.Recurrence
		var date *time.Time

		if err := rows.Scan(&rec.ID, &rec.TaskID, &rec.Type, &rec.Interval, &rec.DayOfMonth, &rec.EvenOdd, &rec.NextRunAt, &date); err != nil {
			return nil, err
		}

		if _, exists := recurrenceMap[rec.ID]; !exists {
			recurrenceMap[rec.ID] = &rec
		}
		if date != nil {
			recurrenceMap[rec.ID].Dates = append(recurrenceMap[rec.ID].Dates, *date)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var result []taskdomain.Recurrence
	for _, rec := range recurrenceMap {
		result = append(result, *rec)
	}
	return result, nil
}

type recurrenceScanner interface {
	Scan(dest ...any) error
}

func scanRecurrence(scanner recurrenceScanner) (*taskdomain.Recurrence, error) {
	var recurrence taskdomain.Recurrence

	if err := scanner.Scan(
		&recurrence.ID,
		&recurrence.TaskID,
		&recurrence.Type,
		&recurrence.Interval,
		&recurrence.DayOfMonth,
		&recurrence.EvenOdd,
		&recurrence.NextRunAt,
	); err != nil {
		return nil, err
	}

	return &recurrence, nil
}
