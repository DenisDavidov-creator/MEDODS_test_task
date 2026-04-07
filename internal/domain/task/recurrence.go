package task

import "time"

type RecurrenceType string

const (
	TypeInterval      RecurrenceType = "interval"
	TypeDayOfMonth    RecurrenceType = "day_of_month"
	TypeEvenOdd       RecurrenceType = "even_odd"
	TypeSpecificDates RecurrenceType = "specific_dates"
)

type EvenOddType string

const (
	TypeEven EvenOddType = "even"
	TypeOdd  EvenOddType = "odd"
)

type Recurrence struct {
	ID         int64          `json:"id"`
	TaskID     int64          `json:"task_id"`
	Type       RecurrenceType `json:"type"`
	Interval   *int64         `json:"interval"`
	DayOfMonth *int64         `json:"day_of_month"`
	EvenOdd    *EvenOddType   `json:"even_odd"`
	NextRunAt  *time.Time     `json:"next_run_at"`
}

type RecurrenceDates struct {
	ID           int64     `json:"id"`
	RecurrenceID int64     `json:"recurrence_id"`
	Date         time.Time `json:"date"`
}

func (s RecurrenceType) Valid() bool {
	switch s {
	case TypeInterval, TypeDayOfMonth, TypeEvenOdd, TypeSpecificDates:
		return true
	default:
		return false
	}
}

func (s EvenOddType) Valid() bool {
	switch s {
	case TypeEven, TypeOdd:
		return true
	default:
		return false
	}
}
