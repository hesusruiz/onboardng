package maintenance

import (
	"context"
	"log/slog"
	"slices"
	"time"
)

// EveryHour indicates that the task should run every hour at the specified minute.
const EveryHour = -1

// Schedule defines when a maintenance task should run.
type Schedule struct {
	// Weekdays is a list of days when the task should run.
	// If empty, the task runs every day.
	Weekdays []time.Weekday
	// Hour is the hour of the day (0-23), or EveryHour (< 0) to run every hour.
	Hour int
	// Minute is the minute of the hour (0-59).
	Minute int
}

// Next returns the earliest scheduled time strictly after the given reference time.
func (s Schedule) Next(from time.Time) time.Time {
	// Hourly schedule: runs every hour at Minute
	if s.Hour < 0 {
		candidate := time.Date(from.Year(), from.Month(), from.Day(), from.Hour(), s.Minute, 0, 0, from.Location())
		if !candidate.After(from) {
			candidate = time.Date(candidate.Year(), candidate.Month(), candidate.Day(), candidate.Hour()+1, s.Minute, 0, 0, from.Location())
		}

		// Ensure it falls on an allowed weekday
		if len(s.Weekdays) > 0 {
			for i := 0; i < 7; i++ {
				if slices.Contains(s.Weekdays, candidate.Weekday()) {
					break
				}
				// Advance to the start of the next day at 00:Minute
				candidate = time.Date(candidate.Year(), candidate.Month(), candidate.Day()+1, 0, s.Minute, 0, 0, from.Location())
			}
		}

		return candidate
	}

	// Daily schedule: runs at s.Hour:s.Minute
	candidate := time.Date(from.Year(), from.Month(), from.Day(), s.Hour, s.Minute, 0, 0, from.Location())
	if !candidate.After(from) {
		candidate = candidate.AddDate(0, 0, 1)
	}

	// Advance day by day until we find a matching weekday (at most 7 iterations).
	if len(s.Weekdays) > 0 {
		for i := 0; i < 7; i++ {
			if slices.Contains(s.Weekdays, candidate.Weekday()) {
				break
			}
			candidate = candidate.AddDate(0, 0, 1)
		}
	}

	return candidate
}

// task represents a maintenance task.
type task struct {
	// The name of the task.
	name string
	// The schedule of the task.
	schedule Schedule
	// The function to run for the task.
	fn func(context.Context) error
	// The last time the task was run.
	lastRun time.Time
	// The next time the task is scheduled to run.
	nextRun time.Time
}

// MaintenanceService manages and executes periodic maintenance tasks.
type MaintenanceService struct {
	tasks []*task
}

// NewMaintenanceService creates a new MaintenanceService.
func NewMaintenanceService() *MaintenanceService {
	return &MaintenanceService{}
}

// AddTask registers a new maintenance task with a scheduling policy.
func (s *MaintenanceService) AddTask(name string, schedule Schedule, fn func(context.Context) error) {
	now := time.Now()
	s.tasks = append(s.tasks, &task{
		name:     name,
		schedule: schedule,
		fn:       fn,
		nextRun:  schedule.Next(now),
	})
}

// Start runs the maintenance loop in the background.
func (s *MaintenanceService) Start() {
	slog.Info("Starting maintenance service")
	go s.run()
}

func (s *MaintenanceService) run() {
	// Check every 30 seconds to ensure tasks are triggered promptly.
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for t := range ticker.C {
		s.checkAndRunTasks(t)
	}
}

func (s *MaintenanceService) checkAndRunTasks(now time.Time) {
	for _, t := range s.tasks {
		if s.shouldRun(t, now) {
			// Mark as last run immediately and advance next run time to avoid duplicate executions.
			t.lastRun = now
			t.nextRun = t.schedule.Next(now)
			go func(currentTask *task) {
				// Create a context with a 1-hour deadline for the task
				// That means that tasks can not last longer than 1 hour otherwise they will be cancelled.
				taskCtx, cancel := context.WithTimeout(context.Background(), 1*time.Hour)
				defer cancel()

				slog.Info("Running maintenance task", "name", currentTask.name)
				if err := currentTask.fn(taskCtx); err != nil {
					slog.Error("Maintenance task failed", "name", currentTask.name, "error", err)
				}
				slog.Info("Maintenance task completed", "name", currentTask.name)
			}(t)
		}
	}
}

func (s *MaintenanceService) shouldRun(t *task, now time.Time) bool {
	if t.nextRun.IsZero() {
		return false
	}
	return !now.Before(t.nextRun)
}
