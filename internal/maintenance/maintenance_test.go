package maintenance

import (
	"context"
	"testing"
	"time"
)

func TestScheduleNext(t *testing.T) {
	// Everyday at 03:00
	schedEveryday := Schedule{Hour: 3, Minute: 0}

	// Sundays at 04:30
	schedSunday := Schedule{Weekdays: []time.Weekday{time.Sunday}, Hour: 4, Minute: 30}

	// Mon and Thu at 05:00
	schedMonThu := Schedule{Weekdays: []time.Weekday{time.Monday, time.Thursday}, Hour: 5, Minute: 0}

	tests := []struct {
		name     string
		schedule Schedule
		from     time.Time
		want     time.Time
	}{
		{
			name:     "Everyday - from earlier today",
			schedule: schedEveryday,
			from:     time.Date(2026, 2, 28, 1, 0, 0, 0, time.UTC),
			want:     time.Date(2026, 2, 28, 3, 0, 0, 0, time.UTC),
		},
		{
			name:     "Everyday - from exact scheduled time advances to tomorrow",
			schedule: schedEveryday,
			from:     time.Date(2026, 2, 28, 3, 0, 0, 0, time.UTC),
			want:     time.Date(2026, 3, 1, 3, 0, 0, 0, time.UTC),
		},
		{
			name:     "Everyday - from later today advances to tomorrow",
			schedule: schedEveryday,
			from:     time.Date(2026, 2, 28, 4, 0, 0, 0, time.UTC),
			want:     time.Date(2026, 3, 1, 3, 0, 0, 0, time.UTC),
		},
		{
			name:     "Sunday - from Saturday advances to Sunday",
			schedule: schedSunday,
			from:     time.Date(2026, 2, 28, 10, 0, 0, 0, time.UTC), // Saturday
			want:     time.Date(2026, 3, 1, 4, 30, 0, 0, time.UTC),  // Sunday
		},
		{
			name:     "Sunday - from Sunday earlier",
			schedule: schedSunday,
			from:     time.Date(2026, 3, 1, 2, 0, 0, 0, time.UTC),  // Sunday
			want:     time.Date(2026, 3, 1, 4, 30, 0, 0, time.UTC), // Sunday
		},
		{
			name:     "Sunday - from Sunday after advances to next Sunday",
			schedule: schedSunday,
			from:     time.Date(2026, 3, 1, 5, 0, 0, 0, time.UTC),  // Sunday
			want:     time.Date(2026, 3, 8, 4, 30, 0, 0, time.UTC), // Next Sunday
		},
		{
			name:     "MonThu - from Sunday advances to Monday",
			schedule: schedMonThu,
			from:     time.Date(2026, 2, 22, 12, 0, 0, 0, time.UTC), // Sunday
			want:     time.Date(2026, 2, 23, 5, 0, 0, 0, time.UTC),  // Monday
		},
		{
			name:     "MonThu - from Monday after advances to Thursday",
			schedule: schedMonThu,
			from:     time.Date(2026, 2, 23, 6, 0, 0, 0, time.UTC), // Monday
			want:     time.Date(2026, 2, 26, 5, 0, 0, 0, time.UTC), // Thursday
		},
		{
			name:     "MonThu - from Wednesday advances to Thursday",
			schedule: schedMonThu,
			from:     time.Date(2026, 2, 25, 12, 0, 0, 0, time.UTC), // Wednesday
			want:     time.Date(2026, 2, 26, 5, 0, 0, 0, time.UTC),  // Thursday
		},
		{
			name:     "MonThu - from Thursday after advances to next Monday",
			schedule: schedMonThu,
			from:     time.Date(2026, 2, 26, 6, 0, 0, 0, time.UTC), // Thursday
			want:     time.Date(2026, 3, 2, 5, 0, 0, 0, time.UTC),  // Next Monday
		},
		{
			name:     "EveryHour everyday - earlier in current hour",
			schedule: Schedule{Hour: EveryHour, Minute: 15},
			from:     time.Date(2026, 2, 28, 10, 10, 0, 0, time.UTC),
			want:     time.Date(2026, 2, 28, 10, 15, 0, 0, time.UTC),
		},
		{
			name:     "EveryHour everyday - exact minute advances to next hour",
			schedule: Schedule{Hour: EveryHour, Minute: 15},
			from:     time.Date(2026, 2, 28, 10, 15, 0, 0, time.UTC),
			want:     time.Date(2026, 2, 28, 11, 15, 0, 0, time.UTC),
		},
		{
			name:     "EveryHour everyday - later in current hour advances to next hour",
			schedule: Schedule{Hour: EveryHour, Minute: 15},
			from:     time.Date(2026, 2, 28, 10, 20, 0, 0, time.UTC),
			want:     time.Date(2026, 2, 28, 11, 15, 0, 0, time.UTC),
		},
		{
			name:     "EveryHour everyday - late night advances across midnight",
			schedule: Schedule{Hour: EveryHour, Minute: 15},
			from:     time.Date(2026, 2, 28, 23, 45, 0, 0, time.UTC),
			want:     time.Date(2026, 3, 1, 0, 15, 0, 0, time.UTC),
		},
		{
			name:     "EveryHour on Sunday - from Saturday advances to Sunday 00:Minute",
			schedule: Schedule{Weekdays: []time.Weekday{time.Sunday}, Hour: -1, Minute: 30},
			from:     time.Date(2026, 2, 28, 14, 0, 0, 0, time.UTC), // Saturday
			want:     time.Date(2026, 3, 1, 0, 30, 0, 0, time.UTC),  // Sunday
		},
		{
			name:     "EveryHour on Sunday - during Sunday runs next hour",
			schedule: Schedule{Weekdays: []time.Weekday{time.Sunday}, Hour: EveryHour, Minute: 30},
			from:     time.Date(2026, 3, 1, 14, 15, 0, 0, time.UTC), // Sunday
			want:     time.Date(2026, 3, 1, 14, 30, 0, 0, time.UTC), // Sunday
		},
		{
			name:     "EveryHour on Sunday - end of Sunday advances to next Sunday 00:Minute",
			schedule: Schedule{Weekdays: []time.Weekday{time.Sunday}, Hour: EveryHour, Minute: 30},
			from:     time.Date(2026, 3, 1, 23, 45, 0, 0, time.UTC), // Sunday
			want:     time.Date(2026, 3, 8, 0, 30, 0, 0, time.UTC),  // Next Sunday
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.schedule.Next(tt.from)
			if !got.Equal(tt.want) {
				t.Errorf("Next() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldRun(t *testing.T) {
	s := NewMaintenanceService()

	targetTime := time.Date(2026, 2, 28, 3, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		nextRun time.Time
		now     time.Time
		want    bool
	}{
		{
			name:    "Before scheduled time",
			nextRun: targetTime,
			now:     targetTime.Add(-10 * time.Second),
			want:    false,
		},
		{
			name:    "Exactly at scheduled time",
			nextRun: targetTime,
			now:     targetTime,
			want:    true,
		},
		{
			name:    "After scheduled time (ticker drift/delay)",
			nextRun: targetTime,
			now:     targetTime.Add(45 * time.Second),
			want:    true,
		},
		{
			name:    "Significantly after scheduled time (paused/delayed tick)",
			nextRun: targetTime,
			now:     targetTime.Add(2 * time.Minute),
			want:    true,
		},
		{
			name:    "Uninitialized task (zero nextRun)",
			nextRun: time.Time{},
			now:     targetTime,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &task{
				nextRun: tt.nextRun,
			}
			if got := s.shouldRun(task, tt.now); got != tt.want {
				t.Errorf("shouldRun() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTaskExecution(t *testing.T) {
	s := NewMaintenanceService()

	executed := make(chan struct{})
	calledWithContext := false
	hasDeadline := false

	s.AddTask("test-task", Schedule{Hour: 10, Minute: 0}, func(ctx context.Context) error {
		calledWithContext = true
		deadline, ok := ctx.Deadline()
		if ok {
			hasDeadline = true
			remaining := time.Until(deadline)
			if remaining > 59*time.Minute && remaining < 61*time.Minute {
				// Success
			} else {
				t.Errorf("expected deadline ~1h, got %v", remaining)
			}
		}
		close(executed)
		return nil
	})

	now := time.Date(2026, 2, 28, 10, 0, 15, 0, time.UTC) // 15 seconds after 10:00 (simulate tick)
	// Ensure the task's nextRun is set to 10:00:00 on this test date
	s.tasks[0].nextRun = time.Date(2026, 2, 28, 10, 0, 0, 0, time.UTC)

	// Trigger task check
	s.checkAndRunTasks(now)

	// Verify that nextRun was advanced and lastRun was updated
	expectedNext := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	if !s.tasks[0].nextRun.Equal(expectedNext) {
		t.Errorf("expected nextRun to advance to %v, got %v", expectedNext, s.tasks[0].nextRun)
	}
	if !s.tasks[0].lastRun.Equal(now) {
		t.Errorf("expected lastRun to be %v, got %v", now, s.tasks[0].lastRun)
	}

	// Wait for task goroutine to execute
	select {
	case <-executed:
		if !calledWithContext {
			t.Error("task not called with context")
		}
		if !hasDeadline {
			t.Error("context has no deadline")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("task did not execute in time")
	}
}
