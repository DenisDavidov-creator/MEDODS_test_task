package task

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type Service struct {
	repo           Repository
	recurrenceRepo RecurrenceRepository
	trans          TransactorInterface
	now            func() time.Time
	logger         *slog.Logger
}

func NewService(repo Repository, recurrenceRepo RecurrenceRepository, trans TransactorInterface, logger *slog.Logger) *Service {
	return &Service{
		repo:           repo,
		recurrenceRepo: recurrenceRepo,
		now:            func() time.Time { return time.Now().UTC() },
		trans:          trans,
		logger:         logger,
	}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (*taskdomain.Task, error) {
	normalized, err := validateCreateInput(input)
	if err != nil {
		return nil, err
	}

	modelTask := &taskdomain.Task{
		Title:       normalized.Title,
		Description: normalized.Description,
		Status:      normalized.Status,
	}
	now := s.now()
	modelTask.CreatedAt = now
	modelTask.UpdatedAt = now

	var modelRecurrence *taskdomain.Recurrence
	if input.Recurrence != nil {
		modelRecurrence = &taskdomain.Recurrence{
			Type:       input.Recurrence.Type,
			Interval:   input.Recurrence.Interval,
			DayOfMonth: input.Recurrence.DayOfMonth,
			EvenOdd:    input.Recurrence.EvenOdd,
			Dates:      input.Recurrence.Dates,
		}

		nextRunAt, err := calculateNextRunAt(modelRecurrence, s.now())
		if err != nil {
			return nil, err
		}
		modelRecurrence.NextRunAt = nextRunAt
	}

	var created = &taskdomain.Task{}
	err = s.trans.WithinTransaction(ctx, func(ctx context.Context) error {
		created, err = s.repo.Create(ctx, modelTask)
		if err != nil {
			return err
		}

		if modelRecurrence != nil {
			modelRecurrence.TaskID = created.ID
			createdRecurrence, err := s.recurrenceRepo.Create(ctx, modelRecurrence)
			if err != nil {
				return err
			}
			created.Recurrence = createdRecurrence
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (s *Service) GetByID(ctx context.Context, id int64) (*taskdomain.Task, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}
	task, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	recurrence, err := s.recurrenceRepo.GetByTaskID(ctx, task.ID)
	if err != nil && !errors.Is(err, taskdomain.ErrNotFound) {
		return nil, err
	}
	task.Recurrence = recurrence
	return task, nil
}

func (s *Service) Update(ctx context.Context, id int64, input UpdateInput) (*taskdomain.Task, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	normalized, err := validateUpdateInput(input)
	if err != nil {
		return nil, err
	}

	var modelRecurrence *taskdomain.Recurrence
	if input.Recurrence != nil {
		modelRecurrence = &taskdomain.Recurrence{
			Type:       input.Recurrence.Type,
			Interval:   input.Recurrence.Interval,
			DayOfMonth: input.Recurrence.DayOfMonth,
			EvenOdd:    input.Recurrence.EvenOdd,
			Dates:      input.Recurrence.Dates,
		}

		nextRunAt, err := calculateNextRunAt(modelRecurrence, s.now())
		if err != nil {
			return nil, err
		}
		modelRecurrence.NextRunAt = nextRunAt
	}

	model := &taskdomain.Task{
		ID:          id,
		Title:       normalized.Title,
		Description: normalized.Description,
		Status:      normalized.Status,
		UpdatedAt:   s.now(),
	}
	var updated = &taskdomain.Task{}
	err = s.trans.WithinTransaction(ctx, func(ctx context.Context) error {
		updated, err = s.repo.Update(ctx, model)
		if err != nil {
			return err
		}

		existing, err := s.recurrenceRepo.GetByTaskID(ctx, id)
		if err != nil && !errors.Is(err, taskdomain.ErrNotFound) {
			return err
		}

		switch {
		case existing != nil && modelRecurrence != nil:
			modelRecurrence.TaskID = id
			modelRecurrence.ID = existing.ID
			updated.Recurrence, err = s.recurrenceRepo.Update(ctx, modelRecurrence)
		case existing != nil && modelRecurrence == nil:
			err = s.recurrenceRepo.Delete(ctx, existing.ID)
		case existing == nil && modelRecurrence != nil:
			modelRecurrence.TaskID = id
			updated.Recurrence, err = s.recurrenceRepo.Create(ctx, modelRecurrence)
		}
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	return s.repo.Delete(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]taskdomain.Task, error) {

	tasks, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(tasks))
	for _, task := range tasks {
		ids = append(ids, task.ID)
	}

	recurrences, err := s.recurrenceRepo.ListByTaskIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	recurrenceMap := map[int64]*taskdomain.Recurrence{}
	for i := range recurrences {
		recurrenceMap[recurrences[i].TaskID] = &recurrences[i]
	}

	for i := range tasks {
		tasks[i].Recurrence = recurrenceMap[tasks[i].ID]
	}

	return tasks, nil
}

func validateCreateInput(input CreateInput) (CreateInput, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)

	if input.Title == "" {
		return CreateInput{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	if input.Status == "" {
		input.Status = taskdomain.StatusNew
	}

	if !input.Status.Valid() {
		return CreateInput{}, fmt.Errorf("%w: invalid status", ErrInvalidInput)
	}
	err := validateRecurrenceInput(input.Recurrence)
	if err != nil {
		return CreateInput{}, fmt.Errorf("%w", err)
	}

	return input, nil
}

func validateUpdateInput(input UpdateInput) (UpdateInput, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)

	if input.Title == "" {
		return UpdateInput{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	if !input.Status.Valid() {
		return UpdateInput{}, fmt.Errorf("%w: invalid status", ErrInvalidInput)
	}
	err := validateRecurrenceInput(input.Recurrence)
	if err != nil {
		return UpdateInput{}, fmt.Errorf("%w", err)
	}

	return input, nil
}

func validateRecurrenceInput(input *RecurrenceInput) error {
	if input == nil {
		return nil
	}
	if !input.Type.Valid() {
		return fmt.Errorf("%w: invalid recurrence type", ErrInvalidInput)
	}

	switch input.Type {
	case taskdomain.TypeInterval:
		if input.Interval == nil || *input.Interval <= 0 {
			return fmt.Errorf("%w: interval must be positive", ErrInvalidInput)
		}
	case taskdomain.TypeDayOfMonth:
		if input.DayOfMonth == nil || *input.DayOfMonth < 1 || *input.DayOfMonth > 31 {
			return fmt.Errorf("%w: day of month must be between 1 and 31", ErrInvalidInput)
		}
	case taskdomain.TypeEvenOdd:
		if input.EvenOdd == nil || !input.EvenOdd.Valid() {
			return fmt.Errorf("%w: even_odd must be 'even' or 'odd'", ErrInvalidInput)
		}
	case taskdomain.TypeSpecificDates:
		if input.Dates == nil {
			return fmt.Errorf("%w: specific_dates must not be empty", ErrInvalidInput)
		}
		now := time.Now()
		hasFuture := false
		for _, date := range input.Dates {
			if date.After(now) {
				hasFuture = true
				break
			}
		}
		if !hasFuture {
			return fmt.Errorf("%w: specific_dates must have at least one future day", ErrInvalidInput)
		}
	}
	return nil
}

func (s *Service) ProcessDue(ctx context.Context) error {
	recurrences, err := s.recurrenceRepo.ListDue(ctx, s.now())
	if err != nil {
		return fmt.Errorf("Internal error: %w", err)
	}
	for _, value := range recurrences {
		err = s.trans.WithinTransaction(ctx, func(ctx context.Context) error {
			task, err := s.repo.GetByID(ctx, value.TaskID)
			if err != nil {
				return fmt.Errorf("task doesn't exist: %w", err)
			}
			_, err = s.repo.Create(ctx, task)
			if err != nil {
				return fmt.Errorf("task doesn't created: %w", err)
			}
			nextTime, err := calculateNextRunAt(&value, s.now())
			if err != nil {
				return fmt.Errorf("doesn't recieve nextTime: %w", err)
			}
			if nextTime == nil {
				return s.recurrenceRepo.Delete(ctx, value.ID)
			}
			value.NextRunAt = nextTime
			_, err = s.recurrenceRepo.Update(ctx, &value)
			if err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			s.logger.Error("Doesn't create task",
				slog.Int64("ID", value.ID),
				slog.String("error", err.Error()))
		}
	}
	return nil
}

func calculateNextRunAt(recurrence *taskdomain.Recurrence, from time.Time) (*time.Time, error) {
	switch recurrence.Type {
	case taskdomain.TypeInterval:
		next := from.AddDate(0, 0, int(*recurrence.Interval))
		return &next, nil
	case taskdomain.TypeDayOfMonth:
		day := int(*recurrence.DayOfMonth)
		year, month, _ := from.Date()
		targetMonth := month
		if int64(from.Day()) >= *recurrence.DayOfMonth {
			targetMonth++
		}
		lastDayInMonth := time.Date(year, targetMonth+1, 0, 0, 0, 0, 0, from.Location()).Day()
		actualDay := day
		if actualDay > lastDayInMonth {
			actualDay = lastDayInMonth
		}
		next := time.Date(year, targetMonth, actualDay, 0, 0, 0, 0, from.Location())
		return &next, nil
	case taskdomain.TypeEvenOdd:
		_, _, day := from.Date()
		isEven := day%2 == 0
		wantsEven := *recurrence.EvenOdd == taskdomain.TypeEven
		if isEven == wantsEven {
			next := from.AddDate(0, 0, 2)
			return &next, nil
		}
		next := from.AddDate(0, 0, 1)
		return &next, nil
	case taskdomain.TypeSpecificDates:
		var minDate time.Time
		for _, date := range recurrence.Dates {
			if date.After(from) && (minDate.IsZero() || date.Before(minDate)) {
				minDate = date
			}
		}
		if minDate.IsZero() {
			return nil, nil
		}
		return &minDate, nil
	}
	return nil, fmt.Errorf("unexpected recurrence type")
}
