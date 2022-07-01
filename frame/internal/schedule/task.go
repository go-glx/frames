package schedule

import "time"

type (
	Task struct {
		priority         Priority      // task schedule priority against another tasks
		runAtLeastOnceIn time.Duration // but anyway it SHOULD be executed at least once per X time
		runAtMostOnceIn  time.Duration // do not run it too often
		taskFn           taskFn

		// stats
		currentPriority float32 // -1; [0..1]; +2
		lastRunAt       time.Time
		avgDuration     time.Duration
		runsCount       uint64
	}

	taskFn = func()
)

func NewTask(
	fn taskFn,
	priority Priority,
	runAtLeastOnceIn time.Duration,
	runAtMostOnceIn time.Duration,
) *Task {
	return &Task{
		priority:         priority,
		runAtLeastOnceIn: runAtLeastOnceIn,
		runAtMostOnceIn:  runAtMostOnceIn,
		taskFn:           fn,
	}
}
