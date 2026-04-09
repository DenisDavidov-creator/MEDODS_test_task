package task

import (
	"testing"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
	"github.com/stretchr/testify/assert"
)

func ptr[T any](v T) *T { return &v }

func TestCalculateNextRunAt(t *testing.T) {
	tests := []struct {
		name       string
		recurrence taskdomain.Recurrence
		from       time.Time
		want       time.Time
		wantNil    bool
	}{
		{
			name: "specific_dates: has future date",
			recurrence: taskdomain.Recurrence{
				Type: taskdomain.TypeSpecificDates,
				Dates: []time.Time{
					time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
					time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
					time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			from: time.Date(2026, 4, 9, 0, 0, 0, 0, time.UTC),
			want: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "specific_dates: all dates in past",
			recurrence: taskdomain.Recurrence{
				Type: taskdomain.TypeSpecificDates,
				Dates: []time.Time{
					time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
					time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			from:    time.Date(2026, 4, 9, 0, 0, 0, 0, time.UTC),
			wantNil: true,
		},
		{
			name: "even_odd: even day want odd",
			recurrence: taskdomain.Recurrence{
				Type:    taskdomain.TypeEvenOdd,
				EvenOdd: ptr(taskdomain.TypeOdd),
			},
			from: time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC),
			want: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "even_odd: odd day want even",
			recurrence: taskdomain.Recurrence{
				Type:    taskdomain.TypeEvenOdd,
				EvenOdd: ptr(taskdomain.TypeEven),
			},
			from: time.Date(2026, 4, 29, 0, 0, 0, 0, time.UTC),
			want: time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "even_odd: odd day want odd",
			recurrence: taskdomain.Recurrence{
				Type:    taskdomain.TypeEvenOdd,
				EvenOdd: ptr(taskdomain.TypeOdd),
			},
			from: time.Date(2026, 4, 29, 0, 0, 0, 0, time.UTC),
			want: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "even_odd: even day want even",
			recurrence: taskdomain.Recurrence{
				Type:    taskdomain.TypeEvenOdd,
				EvenOdd: ptr(taskdomain.TypeEven),
			},
			from: time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC),
			want: time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "day_of_month: 31 February",
			recurrence: taskdomain.Recurrence{
				Type:       taskdomain.TypeDayOfMonth,
				DayOfMonth: ptr(int64(31)),
			},
			from: time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC),
			want: time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "day_of_month: Current month",
			recurrence: taskdomain.Recurrence{
				Type:       taskdomain.TypeDayOfMonth,
				DayOfMonth: ptr(int64(20)),
			},
			from: time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC),
			want: time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "day_of_month: Next month",
			recurrence: taskdomain.Recurrence{
				Type:       taskdomain.TypeDayOfMonth,
				DayOfMonth: ptr(int64(20)),
			},
			from: time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC),
			want: time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "interval 3 days",
			recurrence: taskdomain.Recurrence{
				Type:     taskdomain.TypeInterval,
				Interval: ptr(int64(3)),
			},
			from: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
			want: time.Date(2026, 4, 4, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := calculateNextRunAt(&tt.recurrence, tt.from)

			if tt.wantNil {
				assert.Nil(t, got)
				return
			}

			assert.NoError(t, err)

			assert.Equal(t, tt.want, *got)
		})
	}
}
