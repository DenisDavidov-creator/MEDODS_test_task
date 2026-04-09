package task

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type Service struct {
	repo           Repository
	recurrenceRepo RecurrenceRepository
	now            func() time.Time
}

func NewService(repo Repository, recurrenceRepo RecurrenceRepository) *Service {
	return &Service{
		repo:           repo,
		recurrenceRepo: recurrenceRepo,
		now:            func() time.Time { return time.Now().UTC() },
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

	//TODO transaction?
	created, err := s.repo.Create(ctx, modelTask)
	if err != nil {
		return nil, err
	}

	if modelRecurrence != nil {
		modelRecurrence.TaskID = created.ID
		createdRecurrence, err := s.recurrenceRepo.Create(ctx, modelRecurrence)
		if err != nil {
			return nil, err
		}
		created.Recurrence = createdRecurrence
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
	if err != nil {
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

	updated, err := s.repo.Update(ctx, model)
	if err != nil {
		return nil, err
	}

	existing, err := s.recurrenceRepo.GetByTaskID(ctx, id)
	if err != nil && !errors.Is(err, taskdomain.ErrNotFound) {
		return nil, err
	}

	switch {
	case existing != nil && modelRecurrence != nil:
		modelRecurrence.TaskID = id
		updated.Recurrence, err = s.recurrenceRepo.Update(ctx, modelRecurrence)
	case existing != nil && modelRecurrence == nil:
		err = s.recurrenceRepo.Delete(ctx, existing.ID)
	case existing == nil && modelRecurrence != nil:
		modelRecurrence.TaskID = id
		updated.Recurrence, err = s.recurrenceRepo.Create(ctx, modelRecurrence)
	}
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

	return input, nil
}

func (s *Service) ProcessDue(ctx context.Context) error {
	recurrences, err := s.recurrenceRepo.ListDue(ctx, s.now())
	if err != nil {
		return fmt.Errorf("Internal error: %w", err)
	}
	for _, value := range recurrences {
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
			err := s.recurrenceRepo.Delete(ctx, value.ID)
			if err != nil {
				return err
			}
			continue
		}
		value.NextRunAt = nextTime
		_, err = s.recurrenceRepo.Update(ctx, &value)
		if err != nil {
			return err
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
		if int64(from.Day()) < *recurrence.DayOfMonth {
			lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, from.Location()).Day()
			if day > lastDay {
				day = lastDay
			}
			next := time.Date(year, month, day, 0, 0, 0, 0, from.Location())
			return &next, nil
		}
		next := time.Date(year, month+1, day, 0, 0, 0, 0, from.Location())
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
