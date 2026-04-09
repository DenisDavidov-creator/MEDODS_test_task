package handlers

import (
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type taskMutationDTO struct {
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Status      taskdomain.Status `json:"status"`
	Recurrence  *recurrenceDTO    `json:"recurrence"`
}

type recurrenceDTO struct {
	Type          taskdomain.RecurrenceType `json:"type"`
	Interval      *int64                    `json:"interval,omitempty"`
	DayOfMonth    *int64                    `json:"day_of_month,omitempty"`
	EvenOdd       *taskdomain.EvenOddType   `json:"even_odd,omitempty"`
	SpecificDates []time.Time               `json:"specific_dates,omitempty"`
}

type taskDTO struct {
	ID          int64             `json:"id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Status      taskdomain.Status `json:"status"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Recurrence  *recurrenceDTO    `json:"recurrence"`
}

func newTaskDTO(task *taskdomain.Task) taskDTO {
	dto := taskDTO{
		ID:          task.ID,
		Title:       task.Title,
		Description: task.Description,
		Status:      task.Status,
		CreatedAt:   task.CreatedAt,
		UpdatedAt:   task.UpdatedAt,
	}
	if task.Recurrence != nil {
		dto.Recurrence = &recurrenceDTO{
			Type:          task.Recurrence.Type,
			Interval:      task.Recurrence.Interval,
			DayOfMonth:    task.Recurrence.DayOfMonth,
			EvenOdd:       task.Recurrence.EvenOdd,
			SpecificDates: task.Recurrence.Dates,
		}
	}

	return dto
}
